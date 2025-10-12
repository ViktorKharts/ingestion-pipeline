package cmd

import (
	"context"
	"fmt"
	"log"

	"injestion-pipeline/auth"
	"injestion-pipeline/ingestion"
	"injestion-pipeline/storage"

	"github.com/spf13/cobra"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	folderID string
)

var ingestCmd = &cobra.Command{
	Use:   "ingest [folder-id]",
	Short: "Ingest documents from Google Drive folder",
	Long: `Recursively traverse a Google Drive folder and ingest all .txt and .md files.

The folder ID can be found in the Google Drive URL:
https://drive.google.com/drive/folders/FOLDER_ID_HERE`,
	Args: cobra.MaximumNArgs(1),
	RunE: runIngest,
}

func init() {
	ingestCmd.Flags().StringVarP(&folderID, "folder", "f", "", "Google Drive folder ID")
}

func runIngest(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	db := storage.NewSQLiteDB(DEFAULT_DB_PATH)
	if err := db.Initialize(); err != nil {
		return fmt.Errorf("Failed to initialize database: %w", err)
	}
	defer db.Close()

	authenticator, err := auth.NewGoogleAuthenticator(auth.Config{
		CredentialsPath: credentialsPath,
		TokenPath:       tokenPath,
		Scopes:          []string{drive.DriveReadonlyScope},
	})
	if err != nil {
		return fmt.Errorf("Failed to instantiate authenticator: %w", err)
	}

	client, err := authenticator.GetHTTPClient(ctx)
	if err != nil {
		return fmt.Errorf("Failed to authenticate: %w", err)
	}

	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("Unable to retrieve Drive client: %w", err)
	}

	if folderID == "" {
		folderID = INGESTION_PIPELINE_FOLDER_ID
	}

	di := ingestion.NewDriveIngester(service)

	documents, err := di.IngestFolder(folderID, "/")
	if err != nil {
		return fmt.Errorf("Failed to ingest folder '%s': %w\n", folderID, err)
	}

	log.Printf("INFO: Saving %d documents to database...\n", len(documents))
	saved := 0
	for _, document := range documents {
		err := db.SaveDocument(ctx, document)
		if err != nil {
			return fmt.Errorf("Warning: Failed to save document %s: %w\n", document.FileName, err)
		} else {
			saved++
			log.Printf("✓ Saved: %s\n", document.FilePath)
		}
	}

	log.Printf("Ingestion complete! Saved %d/%d documents to database.\n", saved, len(documents))
	log.Printf("Use './pipeline search <query>' to search the knowledge base\n")
	return nil
}
