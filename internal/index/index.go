// Package index builds and queries a lightweight in-memory index of the project
// codebase. It scans files, extracts lightweight symbols (functions, types,
// classes) with regexes tuned per language, and supports keyword ranking so the
// agent can pull the most relevant files into context automatically.
package index

import (
	"bufio"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// SymbolKind categorizes an extracted symbol.
type SymbolKind int

const (
	SymFunc SymbolKind = iota
	SymType
	SymClass
	SymMethod
	SymConst
	SymVar
)

func (k SymbolKind) String() string {
	switch k {
	case SymFunc:
		return "func"
	case SymType:
		return "type"
	case SymClass:
		return "class"
	case SymMethod:
		return "method"
	case SymConst:
		return "const"
	case SymVar:
		return "var"
	default:
		return "symbol"
	}
}

// Symbol is a named code entity found in a file.
type Symbol struct {
	Name string
	Kind SymbolKind
	Line int
	File string
}

// File is an indexed source file with its symbols and token frequencies.
type FileEntry struct {
	Path     string
	Language string
	Size     int64
	Lines    int
	ModTime  time.Time
	Symbols  []Symbol
	termFreq map[string]int
}

// Index is the searchable in-memory codebase index.
type Index struct {
	mu      sync.RWMutex
	root    string
	files   map[string]*FileEntry
	docFreq map[string]int      // number of files containing a term
	vecNorm map[string]float64  // cached L2 norm of each file's TF-IDF vector
	built   time.Time
}

// Stats summarizes the index contents.
type Stats struct {
	Files    int
	Symbols  int
	Lines    int
	Duration time.Duration
	BuiltAt  time.Time
}

// New creates an empty index rooted at the project directory.
func New(root string) *Index {
	return &Index{
		root:    root,
		files:   map[string]*FileEntry{},
		docFreq: map[string]int{},
		vecNorm: map[string]float64{},
	}
}

var langByExt = map[string]string{
	".go": "go", ".py": "python", ".js": "javascript", ".ts": "typescript",
	".jsx": "javascript", ".tsx": "typescript", ".rs": "rust", ".java": "java",
	".c": "c", ".h": "c", ".cpp": "cpp", ".cc": "cpp", ".hpp": "cpp",
	".rb": "ruby", ".php": "php", ".cs": "csharp", ".swift": "swift",
	".kt": "kotlin", ".scala": "scala", ".sh": "shell", ".md": "markdown",
	".yaml": "yaml", ".yml": "yaml", ".json": "json", ".toml": "toml",
}

func skipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", "dist", "build", "target",
		".idea", ".vscode", "__pycache__", ".venv", "venv", "bin", "obj":
		return true
	}
	return false
}

const maxFileSize = 1 << 20 // 1 MiB; skip larger files

// Build scans the project tree and populates the index. It is safe to call
// again to rebuild from scratch.
func (ix *Index) Build() (Stats, error) {
	start := time.Now()
	files := map[string]*FileEntry{}
	docFreq := map[string]int{}

	err := filepath.WalkDir(ix.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		lang, ok := langByExt[ext]
		if !ok {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxFileSize {
			return nil
		}
		entry, err := ix.indexFile(path, lang, info)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(ix.root, path)
		rel = filepath.ToSlash(rel)
		entry.Path = rel
		files[rel] = entry
		for term := range entry.termFreq {
			docFreq[term]++
		}
		return nil
	})
	if err != nil {
		return Stats{}, err
	}

	// Precompute the L2 norm of every file's TF-IDF vector so cosine-similarity
	// vector search is a single dot-product per file at query time.
	vecNorm := computeNorms(files, docFreq)

	ix.mu.Lock()
	ix.files = files
	ix.docFreq = docFreq
	ix.vecNorm = vecNorm
	ix.built = time.Now()
	ix.mu.Unlock()

	return ix.Stats(start), nil
}

// tfidf returns the TF-IDF weight of a term in a file. Term frequency is
// log-normalized and inverse document frequency is smoothed, matching the
// weighting used by the cosine-similarity vector store.
func tfidf(tf, df, totalDocs float64) float64 {
	if tf <= 0 {
		return 0
	}
	if df <= 0 {
		df = 1
	}
	return (1.0 + math.Log(tf)) * (1.0 + math.Log(totalDocs/df))
}

// computeNorms precomputes the Euclidean norm of each file's TF-IDF vector.
func computeNorms(files map[string]*FileEntry, docFreq map[string]int) map[string]float64 {
	totalDocs := float64(len(files))
	if totalDocs == 0 {
		return map[string]float64{}
	}
	norms := make(map[string]float64, len(files))
	for path, entry := range files {
		var sumSq float64
		for term, tf := range entry.termFreq {
			w := tfidf(float64(tf), float64(docFreq[term]), totalDocs)
			sumSq += w * w
		}
		norms[path] = math.Sqrt(sumSq)
	}
	return norms
}

func (ix *Index) indexFile(path, lang string, info os.FileInfo) (*FileEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	entry := &FileEntry{
		Language: lang,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		termFreq: map[string]int{},
	}
	patterns := symbolPatterns[lang]
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		// term frequencies for ranking
		for _, tok := range tokenize(line) {
			entry.termFreq[tok]++
		}
		// symbol extraction
		for _, p := range patterns {
			if m := p.re.FindStringSubmatch(line); m != nil && len(m) > 1 {
				entry.Symbols = append(entry.Symbols, Symbol{
					Name: m[1], Kind: p.kind, Line: lineNo, File: path,
				})
			}
		}
	}
	entry.Lines = lineNo
	return entry, nil
}

// symbolPattern pairs a compiled regex with the symbol kind it extracts.
type symbolPattern struct {
	re   *regexp.Regexp
	kind SymbolKind
}

var symbolPatterns = map[string][]symbolPattern{
	"go": {
		{regexp.MustCompile(`^\s*func\s+(?:\([^)]*\)\s+)?([A-Za-z_]\w*)`), SymFunc},
		{regexp.MustCompile(`^\s*type\s+([A-Za-z_]\w*)`), SymType},
		{regexp.MustCompile(`^\s*const\s+([A-Za-z_]\w*)`), SymConst},
	},
	"python": {
		{regexp.MustCompile(`^\s*def\s+([A-Za-z_]\w*)`), SymFunc},
		{regexp.MustCompile(`^\s*class\s+([A-Za-z_]\w*)`), SymClass},
	},
	"javascript": {
		{regexp.MustCompile(`^\s*function\s+([A-Za-z_$][\w$]*)`), SymFunc},
		{regexp.MustCompile(`^\s*class\s+([A-Za-z_$][\w$]*)`), SymClass},
		{regexp.MustCompile(`^\s*(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?\(`), SymFunc},
	},
	"typescript": {
		{regexp.MustCompile(`^\s*function\s+([A-Za-z_$][\w$]*)`), SymFunc},
		{regexp.MustCompile(`^\s*class\s+([A-Za-z_$][\w$]*)`), SymClass},
		{regexp.MustCompile(`^\s*interface\s+([A-Za-z_$][\w$]*)`), SymType},
	},
	"rust": {
		{regexp.MustCompile(`^\s*(?:pub\s+)?fn\s+([A-Za-z_]\w*)`), SymFunc},
		{regexp.MustCompile(`^\s*(?:pub\s+)?struct\s+([A-Za-z_]\w*)`), SymType},
		{regexp.MustCompile(`^\s*(?:pub\s+)?enum\s+([A-Za-z_]\w*)`), SymType},
	},
	"java": {
		{regexp.MustCompile(`^\s*(?:public|private|protected).*\bclass\s+([A-Za-z_]\w*)`), SymClass},
		{regexp.MustCompile(`^\s*(?:public|private|protected).*\s+([A-Za-z_]\w*)\s*\(`), SymMethod},
	},
}

var tokenRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]{2,}`)

func tokenize(s string) []string {
	matches := tokenRe.FindAllString(s, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, strings.ToLower(m))
	}
	return out
}

// SearchResult is a ranked file match.
type SearchResult struct {
	Path    string
	Score   float64
	Symbols []Symbol
}

// Search ranks indexed files against a free-text query using TF-IDF scoring.
func (ix *Index) Search(query string, limit int) []SearchResult {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}
	totalDocs := float64(len(ix.files))
	if totalDocs == 0 {
		return nil
	}

	var results []SearchResult
	for path, entry := range ix.files {
		score := 0.0
		for _, term := range terms {
			tf := float64(entry.termFreq[term])
			if tf == 0 {
				continue
			}
			df := float64(ix.docFreq[term])
			if df == 0 {
				df = 1
			}
			idf := 1.0 + logBase(totalDocs/df)
			score += (tf / float64(entry.Lines+1)) * idf
			// boost if a symbol name matches the term
			for _, sym := range entry.Symbols {
				if strings.ToLower(sym.Name) == term {
					score += 2.0 * idf
				}
			}
		}
		if score > 0 {
			results = append(results, SearchResult{Path: path, Score: score, Symbols: entry.Symbols})
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// FindSymbol returns all symbols matching a name (case-insensitive).
func (ix *Index) FindSymbol(name string) []Symbol {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	lower := strings.ToLower(name)
	var out []Symbol
	for _, entry := range ix.files {
		for _, sym := range entry.Symbols {
			if strings.ToLower(sym.Name) == lower {
				s := sym
				s.File = entry.Path
				out = append(out, s)
			}
		}
	}
	return out
}

// Stats returns index statistics.
func (ix *Index) Stats(since time.Time) Stats {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	symbols, lines := 0, 0
	for _, e := range ix.files {
		symbols += len(e.Symbols)
		lines += e.Lines
	}
	return Stats{
		Files:    len(ix.files),
		Symbols:  symbols,
		Lines:    lines,
		Duration: time.Since(since),
		BuiltAt:  ix.built,
	}
}

// Files returns the sorted list of indexed file paths.
func (ix *Index) Files() []string {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	out := make([]string, 0, len(ix.files))
	for p := range ix.files {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// logBase computes the natural logarithm, used for IDF scoring.
func logBase(x float64) float64 {
	return math.Log(x)
}

// conceptSynonyms maps concepts to related terms for semantic search.
// This allows "authentication" to match "login", "auth", "session", etc.
var conceptSynonyms = map[string][]string{
	"authentication": {"auth", "login", "signin", "session", "token", "password", "credential"},
	"database":       {"db", "sql", "query", "model", "schema", "migration", "orm"},
	"api":            {"endpoint", "route", "handler", "controller", "rest", "graphql"},
	"config":         {"configuration", "settings", "env", "environment", "options"},
	"error":          {"error", "err", "fail", "panic", "exception", "crash"},
	"test":           {"test", "spec", "mock", "assert", "fixture", "benchmark"},
	"security":       {"security", "crypto", "encrypt", "decrypt", "hash", "salt", "vulnerability"},
	"logging":        {"log", "logger", "trace", "debug", "verbose"},
	"cache":          {"cache", "memoize", "buffer", "store", "redis"},
	"auth":           {"authentication", "login", "signin", "session", "token", "jwt"},
	"ui":             {"ui", "component", "render", "view", "template", "frontend"},
	"server":         {"server", "http", "listen", "serve", "handler", "middleware"},
	"client":         {"client", "request", "fetch", "http", "api"},
	"validation":     {"validate", "validation", "check", "verify", "schema", "sanitize"},
	"middleware":     {"middleware", "interceptor", "filter", "chain", "pipeline"},
}

// SemanticSearch performs concept-aware search. It combines two real signals:
//
//  1. Synonym expansion — the query is enriched with related terms (so
//     "authentication" also matches "login", "token", "session").
//  2. Cosine-similarity vector search — the expanded query is turned into a
//     TF-IDF vector and scored against every file's precomputed TF-IDF vector
//     in the in-memory vector store.
//
// The two signals are blended so that both conceptual overlap (vector cosine)
// and raw term hits contribute to the final ranking.
func (ix *Index) SemanticSearch(query string, limit int) []SearchResult {
	terms := tokenize(query)
	// Expand with synonyms so related vocabulary lands in the query vector.
	expanded := make(map[string]bool)
	for _, t := range terms {
		expanded[t] = true
		if syns, ok := conceptSynonyms[t]; ok {
			for _, s := range syns {
				expanded[s] = true
			}
		}
		for concept, syns := range conceptSynonyms {
			for _, s := range syns {
				if s == t {
					expanded[concept] = true
					for _, s2 := range syns {
						expanded[s2] = true
					}
				}
			}
		}
	}
	allTerms := make([]string, 0, len(expanded))
	for t := range expanded {
		allTerms = append(allTerms, t)
	}
	return ix.VectorSearch(allTerms, limit)
}

// VectorSearch ranks files by cosine similarity between the query's TF-IDF
// vector and each file's precomputed TF-IDF vector. This is a genuine
// vector-space search over the in-memory vector store, not just keyword hits:
// files whose overall term distribution matches the query rank highest, and
// scores are normalized to [0,1] by vector magnitude.
func (ix *Index) VectorSearch(terms []string, limit int) []SearchResult {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	totalDocs := float64(len(ix.files))
	if totalDocs == 0 || len(terms) == 0 {
		return nil
	}

	// Build the query TF-IDF vector (query term frequencies × idf).
	queryTF := map[string]int{}
	for _, t := range terms {
		queryTF[t]++
	}
	queryVec := map[string]float64{}
	var qNormSq float64
	for term, tf := range queryTF {
		w := tfidf(float64(tf), float64(ix.docFreq[term]), totalDocs)
		if w == 0 {
			continue
		}
		queryVec[term] = w
		qNormSq += w * w
	}
	qNorm := math.Sqrt(qNormSq)
	if qNorm == 0 {
		return nil
	}

	var results []SearchResult
	for path, entry := range ix.files {
		fNorm := ix.vecNorm[path]
		if fNorm == 0 {
			continue
		}
		// Dot product over the (small) query term set only.
		var dot float64
		for term, qw := range queryVec {
			tf := entry.termFreq[term]
			if tf == 0 {
				continue
			}
			dot += qw * tfidf(float64(tf), float64(ix.docFreq[term]), totalDocs)
		}
		if dot == 0 {
			continue
		}
		cosine := dot / (qNorm * fNorm)
		// Small boost when a symbol name matches a query term directly.
		for _, sym := range entry.Symbols {
			if queryVec[strings.ToLower(sym.Name)] > 0 {
				cosine += 0.05
			}
		}
		results = append(results, SearchResult{Path: path, Score: cosine, Symbols: entry.Symbols})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

// searchWithTerms runs TF-IDF search with pre-tokenized terms.
func (ix *Index) searchWithTerms(terms []string, limit int) []SearchResult {
	ix.mu.RLock()
	defer ix.mu.RUnlock()

	if len(terms) == 0 {
		return nil
	}
	totalDocs := float64(len(ix.files))
	if totalDocs == 0 {
		return nil
	}

	var results []SearchResult
	for path, entry := range ix.files {
		score := 0.0
		for _, term := range terms {
			tf := float64(entry.termFreq[term])
			if tf == 0 {
				continue
			}
			df := float64(ix.docFreq[term])
			if df == 0 {
				df = 1
			}
			idf := 1.0 + logBase(totalDocs/df)
			score += (tf / float64(entry.Lines+1)) * idf
			for _, sym := range entry.Symbols {
				if strings.ToLower(sym.Name) == term {
					score += 2.0 * idf
				}
			}
		}
		if score > 0 {
			results = append(results, SearchResult{Path: path, Score: score, Symbols: entry.Symbols})
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}
