package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath          string
	credentialsPath string
	tokenPath       string
)

var rootCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Google Drive knowledge ingestion pipeline",
	Long: `A CLI tool to ingest documents from Google Drive and make them searchable.

Supports .txt and .md files with full-text search capabilities.`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "knowledge.db", "Path to SQLite database")
	rootCmd.PersistentFlags().StringVar(&credentialsPath, "credentials", "credentials.json", "Path to Google OAuth credentials")
	rootCmd.PersistentFlags().StringVar(&tokenPath, "token", "token.json", "Path to OAuth token cache")

	rootCmd.AddCommand(ingestCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(clearCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
