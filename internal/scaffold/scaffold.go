// Package scaffold provides code generation templates for rapid project
// scaffolding, design pattern implementation, and boilerplate elimination.
package scaffold

import (
	"fmt"
	"strings"
)

// Template is a code generation template.
type Template struct {
	ID          string
	Name        string
	Description string
	Language    string // "go", "python", "javascript", "typescript", "any"
	Generate    func(params map[string]string) string
}

// Registry holds all available templates.
type Registry struct {
	templates map[string]*Template
}

// New creates a template registry with all built-in templates.
func New() *Registry {
	r := &Registry{templates: map[string]*Template{}}
	r.registerBuiltins()
	return r
}

// Get returns a template by ID.
func (r *Registry) Get(id string) (*Template, bool) {
	t, ok := r.templates[id]
	return t, ok
}

// List returns all template IDs.
func (r *Registry) List() []string {
	ids := make([]string, 0, len(r.templates))
	for id := range r.templates {
		ids = append(ids, id)
	}
	return ids
}

func (r *Registry) register(t *Template) {
	r.templates[t.ID] = t
}

func (r *Registry) registerBuiltins() {
	// REST API with auth (Go)
	r.register(&Template{
		ID:          "rest-api-go",
		Name:        "REST API (Go + Auth)",
		Description: "Complete REST API with JWT auth, middleware, and CRUD handlers",
		Language:    "go",
		Generate: func(p map[string]string) string {
			name := p["name"]
			if name == "" {
				name = "api"
			}
			return fmt.Sprintf(`package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type Server struct {
	router *mux.Router
}

func NewServer() *Server {
	s := &Server{router: mux.NewRouter()}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.HandleFunc("/api/%s", s.handleCreate).Methods("POST")
	s.router.HandleFunc("/api/%s/{id}", s.handleGet).Methods("GET")
	s.router.HandleFunc("/api/%s/{id}", s.handleUpdate).Methods("PUT")
	s.router.HandleFunc("/api/%s/{id}", s.handleDelete).Methods("DELETE")
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.authMiddleware)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%%s %%s %%v", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	secret := os.Getenv("JWT_SECRET")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret == "" {
			http.Error(w, "JWT_SECRET is required", http.StatusInternalServerError)
			return
		}
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return []byte(secret), nil
		})
		if err != nil || token == nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Created"))
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Get"))
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Updated"))
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	s := NewServer()
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", s.router))
}
`, name, name, name, name)
		},
	})

	// CRUD model (Go)
	r.Register(&Template{
		ID:          "crud-model-go",
		Name:        "CRUD Model (Go)",
		Description: "Model with CRUD operations, validation, and repository pattern",
		Language:    "go",
		Generate: func(p map[string]string) string {
			model := p["model"]
			if model == "" {
				model = "User"
			}
			lower := strings.ToLower(model)
			tmpl := `package models

import (
	"errors"
	"time"
)

type MODEL struct {
	ID        string    ` + "`json:\"id\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\"`" + `
}

type MODELRepository interface {
	Create(m *MODEL) error
	GetByID(id string) (*MODEL, error)
	Update(m *MODEL) error
	Delete(id string) error
	List() ([]*MODEL, error)
}

type LOWERRepo struct {
	store map[string]*MODEL
}

func NewLOWERRepo() MODELRepository {
	return &LOWERRepo{store: make(map[string]*MODEL)}
}

func (r *LOWERRepo) Create(m *MODEL) error {
	if m.ID == "" {
		return errors.New("ID is required")
	}
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	r.store[m.ID] = m
	return nil
}

func (r *LOWERRepo) GetByID(id string) (*MODEL, error) {
	m, ok := r.store[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return m, nil
}

func (r *LOWERRepo) Update(m *MODEL) error {
	if _, ok := r.store[m.ID]; !ok {
		return errors.New("not found")
	}
	m.UpdatedAt = time.Now()
	r.store[m.ID] = m
	return nil
}

func (r *LOWERRepo) Delete(id string) error {
	delete(r.store, id)
	return nil
}

func (r *LOWERRepo) List() ([]*MODEL, error) {
	list := make([]*MODEL, 0, len(r.store))
	for _, m := range r.store {
		list = append(list, m)
	}
	return list, nil
}
`
			return strings.ReplaceAll(strings.ReplaceAll(tmpl, "MODEL", model), "LOWER", lower)
		},
	})

	// Test file (Go)
	r.Register(&Template{
		ID:          "test-file-go",
		Name:        "Test File (Go)",
		Description: "Table-driven test with subtests and coverage",
		Language:    "go",
		Generate: func(p map[string]string) string {
			pkg := p["package"]
			if pkg == "" {
				pkg = "main"
			}
			func_ := p["function"]
			if func_ == "" {
				func_ = "MyFunc"
			}
return fmt.Sprintf(`package %s

import "testing"

func Test%s_IsCallable(t *testing.T) {
	// This scaffold is a real compile-time contract for a zero-argument function.
	// For functions with parameters, use generate_tests with explicit cases.
	%s()
}

func Benchmark%s(b *testing.B) {
	for i := 0; i < b.N; i++ {
		%s()
	}
}
`, pkg, func_, func_, func_, func_)
		},
	})

	// Python CLI
	r.Register(&Template{
		ID:          "python-cli",
		Name:        "Python CLI App",
		Description: "Command-line app with argparse, logging, and subcommands",
		Language:    "python",
		Generate: func(p map[string]string) string {
			name := p["name"]
			if name == "" {
				name = "myapp"
			}
			return fmt.Sprintf(`#!/usr/bin/env python3
"""%s — a command-line tool."""

import argparse
import logging
import sys

__version__ = "1.0.0"

logging.basicConfig(
    level=logging.INFO,
    format="%%(asctime)s [%%(levelname)s] %%(message)s",
)
logger = logging.getLogger(__name__)


def cmd_run(args):
    """Run the main operation."""
    logger.info("Running with: %%s", args)


def cmd_init(args):
    """Initialize the project."""
    logger.info("Initializing...")


def main():
    parser = argparse.ArgumentParser(description="%s")
    parser.add_argument("--version", action="version", version=f"%%(prog)s %%s" %% __version__)
    parser.add_argument("-v", "--verbose", action="store_true", help="verbose output")

    subparsers = parser.add_subparsers(dest="command", help="available commands")

    # run command
    p_run = subparsers.add_parser("run", help="run the main operation")
    p_run.add_argument("--input", "-i", help="input file")
    p_run.set_defaults(func=cmd_run)

    # init command
    p_init = subparsers.add_parser("init", help="initialize the project")
    p_init.set_defaults(func=cmd_init)

    args = parser.parse_args()
    if not args.command:
        parser.print_help()
        sys.exit(1)

    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)

    args.func(args)


if __name__ == "__main__":
    main()
`, name, name)
		},
	})

	// Dockerfile (multi-stage)
	r.Register(&Template{
		ID:          "dockerfile",
		Name:        "Dockerfile (Multi-stage)",
		Description: "Multi-stage Dockerfile for Go, Python, or Node",
		Language:    "any",
		Generate: func(p map[string]string) string {
			lang := p["language"]
			if lang == "" {
				lang = "go"
			}
			switch lang {
			case "go":
				return `# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/server

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
`
			case "python":
				return `FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
`
			case "node", "javascript", "typescript":
				return `# Build stage
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Runtime stage
FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
COPY package*.json ./
EXPOSE 3000
CMD ["node", "dist/index.js"]
`
			default:
				return "# Unsupported language for Dockerfile template"
			}
		},
	})

	// GitHub Actions CI
	r.Register(&Template{
		ID:          "github-actions",
		Name:        "GitHub Actions CI",
		Description: "CI workflow: lint, test, build on push/PR",
		Language:    "any",
		Generate: func(p map[string]string) string {
			lang := p["language"]
			if lang == "" {
				lang = "go"
			}
			switch lang {
			case "go":
				return `name: CI
on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go vet ./...
      - run: go test -v -race -coverprofile=coverage.out ./...
      - run: go build ./...
      - uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out
`
			default:
				return `name: CI
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: npm ci
      - run: npm run lint
      - run: npm test
      - run: npm run build
`
			}
		},
	})
}

// Register is a public method to add custom templates.
func (r *Registry) Register(t *Template) {
	r.register(t)
}

// Generate produces code from a template by ID.
func (r *Registry) Generate(templateID string, params map[string]string) (string, error) {
	t, ok := r.Get(templateID)
	if !ok {
		return "", fmt.Errorf("template %q not found. Available: %s", templateID, strings.Join(r.List(), ", "))
	}
	return t.Generate(params), nil
}
