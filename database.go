package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	pipeline "injestion-pipeline/db"
	"injestion-pipeline/models"

	_ "github.com/mattn/go-sqlite3"
)

type Database interface {
	Initialize() error
	SaveDocument(ctx context.Context, doc *models.Document) error
	SearchDocuments(ctx context.Context, query string, limit int) ([]SearchResult, error)
	ListAllDocuments(ctx context.Context) ([]pipeline.Document, error)
	ClearAll(ctx context.Context) error
	Close() error
}

type SearchResult struct {
	Document pipeline.Document
	Snippet  string
}

type SQLiteDB struct {
	db      *sql.DB
	queries *pipeline.Queries
	dbPath  string
}

func NewSQLiteDB(dbPath string) *SQLiteDB {
	return &SQLiteDB{
		dbPath: dbPath,
	}
}

func (s *SQLiteDB) Initialize() error {
	db, err := sql.Open("sqlite3", s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return fmt.Errorf("failed to set WAL mode: %w", err)
	}

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	_, err = db.Exec(string(schema))
	if err != nil {
		if strings.Contains(err.Error(), "fts5") || strings.Contains(err.Error(), "no such module") {
			return fmt.Errorf("SQLite FTS5 is not enabled. Rebuild with: go build -tags 'fts5'")
		}
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	s.db = db
	s.queries = pipeline.New(db)

	return nil
}

func (s *SQLiteDB) SaveDocument(ctx context.Context, doc *models.Document) error {
	_, err := s.queries.CreateDocument(ctx, pipeline.CreateDocumentParams{
		DriveFileID:  doc.DriveFileID,
		Filename:     doc.FileName,
		Filepath:     doc.FilePath,
		Content:      doc.Content,
		Extension:    doc.Extension,
		LastModified: doc.LastModified,
		SizeBytes:    doc.SizeBytes,
	})

	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	return nil
}

func (s *SQLiteDB) SearchDocuments(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	docs, err := s.queries.SearchDocuments(ctx, pipeline.SearchDocumentsParams{
		Query: query,
		Limit: int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	results := make([]SearchResult, 0, len(docs))
	for _, doc := range docs {
		snippet := generateSnippet(doc.Content, query, 150)
		results = append(results, SearchResult{
			Document: doc,
			Snippet:  snippet,
		})
	}

	return results, nil
}

func (s *SQLiteDB) ListAllDocuments(ctx context.Context) ([]pipeline.Document, error) {
	docs, err := s.queries.ListDocuments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	return docs, nil
}

func (s *SQLiteDB) ClearAll(ctx context.Context) error {
	err := s.queries.DeleteAllDocuments(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear documents: %w", err)
	}
	return nil
}

func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func generateSnippet(content string, query string, maxLength int) string {
	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(content)

	queryTerms := strings.Fields(queryLower)
	if len(queryTerms) == 0 {
		if len(content) > maxLength {
			return content[:maxLength] + "..."
		}
		return content
	}

	firstTerm := queryTerms[0]
	index := strings.Index(contentLower, firstTerm)

	if index == -1 {
		if len(content) > maxLength {
			return content[:maxLength] + "..."
		}
		return content
	}

	start := max(index-50, 0)
	end := min(index+len(firstTerm)+100, len(content))
	snippet := content[start:end]

	if start > 0 {
		snippet = "..." + snippet
	}

	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}
