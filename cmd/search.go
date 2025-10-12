package cmd

import (
	"context"
	"fmt"
	"injestion-pipeline/storage"
	"log"
	"strings"

	"github.com/spf13/cobra"
)

var (
	searchLimit int
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search documents by keyword",
	Long: `Performs full-text search across all ingested documents.

Examples:
  pipeline search "login"
  pipeline search "user authentication"
  pipeline search --limit 10 "error handling"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 20, "Maximum number of results")
}

func runSearch(cmd *cobra.Command, args []string) error {
	if len(args) < 3 {
		fmt.Println("Usage:		./pipeline search <query>")
		fmt.Println("Example:	./pipeline search \"login\"")
		return nil
	}

	query := strings.Join(args, " ")
	ctx := context.Background()

	db := storage.NewSQLiteDB(DEFAULT_DB_PATH)
	if err := db.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Printf("Searching for: \"%s\"\n\n", query)

	results, err := db.SearchDocuments(ctx, query, 20)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		fmt.Printf("No results found for \"%s\"\n", query)
		return nil
	}

	fmt.Printf("Found %d result(s):\n\n", len(results))

	for i, result := range results {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("[%d] %s\n", i+1, result.Document.Filename)
		fmt.Printf("Path: %s\n", result.Document.Filepath)
		fmt.Printf("Modified: %s\n", result.Document.LastModified)
		fmt.Printf("Size: %d bytes\n\n", result.Document.SizeBytes)
		fmt.Printf("Snippet:\n%s\n\n", result.Snippet)
	}
}
