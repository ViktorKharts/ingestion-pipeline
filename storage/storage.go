package storage

import (
	"context"
	pipeline "injestion-pipeline/db"
	"injestion-pipeline/models"
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
