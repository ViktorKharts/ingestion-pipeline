package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"injestion-pipeline/storage"

	"github.com/spf13/cobra"
)

var (
	clearForce bool
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all documents from database",
	Long:  "Permanently delete all ingested documents from the database.",
	RunE:  runClear,
}

func init() {
	clearCmd.Flags().BoolVarP(&clearForce, "force", "f", false, "Skip confirmation prompt")
}

func runClear(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if !clearForce {
		fmt.Print("WARNING: Are you sure you want to clear all documents? (yes/no): ")
		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) != "yes" {
			log.Printf("INFO: Cancelled.\n")
			return nil
		}
	}

	db := storage.NewSQLiteDB(DEFAULT_DB_PATH)
	if err := db.Initialize(); err != nil {
		return fmt.Errorf("Failed to initialize database: %w", err)
	}
	defer db.Close()

	err := db.ClearAll(ctx)
	if err != nil {
		return fmt.Errorf("Failed to clear database: %w", err)
	}

	log.Printf("INFO: All documents cleared from database.\n")
	return nil
}
