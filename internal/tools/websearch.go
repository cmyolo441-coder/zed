package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// webSearch is a tool that searches the web and fetches documentation.
type webSearch struct {
	http *http.Client
}

func newWebSearch() *webSearch {
	return &webSearch{
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 15 * time.Second,
			},
		},
	}
}

// NewWebSearch returns a web search tool ready for registration.
func NewWebSearch() Tool { return newWebSearch() }

func (w *webSearch) Name() string { return "web_search" }
func (w *webSearch) Description() string {
	return "Search the web for real-time information, documentation, or solutions. " +
		"Use for: latest API docs, error messages, Stack Overflow answers, " +
		"library usage examples, or any external knowledge. " +
		"Args: {\"query\": \"search terms\", \"fetch\": \"optional URL to fetch full page\"}"
}
func (w *webSearch) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query — be specific for better results.",
			},
			"fetch": map[string]any{
				"type":        "string",
				"description": "Optional: fetch a specific URL's content directly instead of searching.",
			},
		},
		"required": []string{"query"},
	}
}
func (w *webSearch) RequiresApproval() bool { return false }

func (w *webSearch) Execute(ctx context.Context, args string) (string, error) {
	var parsed struct {
		Query string `json:"query"`
		Fetch string `json:"fetch"`
	}
	if err := parseArgs(args, &parsed); err != nil {
		return "", err
	}
	fetchURL := parsed.Fetch
	if fetchURL != "" {
		return w.fetchPage(ctx, fetchURL)
	}
	query := parsed.Query
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	return w.search(ctx, query)
}

// search uses DuckDuckGo's HTML endpoint (no API key required).
func (w *webSearch) search(ctx context.Context, query string) (string, error) {
	u := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := w.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	results := parseDDGResults(string(body))
	if len(results) == 0 {
		return "No results found for: " + query, nil
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔍 Web search: %q\n\n", query))
	for i, r := range results {
		if i >= 8 {
			break
		}
		b.WriteString(fmt.Sprintf("%d. %s\n   %s\n   %s\n\n", i+1, r.Title, r.URL, r.Snippet))
	}
	b.WriteString("\nUse web_search with \"fetch\" URL to get full page content.")
	return b.String(), nil
}

// fetchPage retrieves and extracts text content from a URL.
func (w *webSearch) fetchPage(ctx context.Context, rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "http") {
		rawURL = "https://" + rawURL
	}
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := w.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 500*1024)) // 500KB max
	if err != nil {
		return "", err
	}
	text := extractText(string(body))
	text = strings.TrimSpace(text)
	if len(text) > 8000 {
		text = text[:8000] + "\n…[truncated]"
	}
	return fmt.Sprintf("📄 %s\n\n%s", rawURL, text), nil
}

type ddgResult struct {
	Title, URL, Snippet string
}

func parseDDGResults(html string) []ddgResult {
	var results []ddgResult
	linkRe := regexp.MustCompile(`<a[^>]*class="result__a"[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	snippetRe := regexp.MustCompile(`<a[^>]*class="result__snippet"[^>]*>(.*?)</a>`)
	links := linkRe.FindAllStringSubmatch(html, -1)
	snippets := snippetRe.FindAllStringSubmatch(html, -1)
	for i, m := range links {
		if i >= 10 {
			break
		}
		href := m[1]
		// DDG uses redirect links; extract actual URL.
		if strings.Contains(href, "uddg=") {
			if u, err := url.Parse(href); err == nil {
				if v := u.Query().Get("uddg"); v != "" {
					href = v
				}
			}
		}
		title := stripTags(m[2])
		snippet := ""
		if i < len(snippets) {
			snippet = stripTags(snippets[i][1])
		}
		results = append(results, ddgResult{Title: title, URL: href, Snippet: snippet})
	}
	return results
}

func extractText(html string) string {
	// Remove scripts, styles, and tags.
	html = regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?s)<nav[^>]*>.*?</nav>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?s)<footer[^>]*>.*?</footer>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, " ")
	// Decode common HTML entities — order matters: decode & LAST so we
	// don't corrupt < > " ' which contain '&'.
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&ldquo;", "\"")
	html = strings.ReplaceAll(html, "&rdquo;", "\"")
	html = strings.ReplaceAll(html, "&apos;", "'")
	html = strings.ReplaceAll(html, "&#39;", "'")
	html = strings.ReplaceAll(html, "&amp;", "&")
	// Collapse whitespace.
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")
	return html
}

func stripTags(s string) string {
	return strings.TrimSpace(regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, ""))
}

// Ensure interface compliance.
var _ Tool = (*webSearch)(nil)
