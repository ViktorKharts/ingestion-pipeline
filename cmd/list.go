package cmd

import (
	"context"
	"fmt"
	"injestion-pipeline/storage"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all documents from database",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	db := storage.NewSQLiteDB(DEFAULT_DB_PATH)
	if err := db.Initialize(); err != nil {
		return fmt.Errorf("Failed to initialize database: %w", err)
	}
	defer db.Close()

	docs, err := db.ListAllDocuments(ctx)
	if err != nil {
		return fmt.Errorf("Failed to list documents: %w", err)
	}

	fmt.Printf("INFO: Total documents in database: %d\n\n", len(docs))

	for i, doc := range docs {
		fmt.Printf("%d. %s (%s) - %d bytes\n", i+1, doc.Filepath, doc.Extension, doc.SizeBytes)
	}

	return nil
}
