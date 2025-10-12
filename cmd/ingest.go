package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"

	"injestion-pipeline/auth"
	"injestion-pipeline/models"
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

	documents, err := ingestFolder(service, folderID, "/")
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

func ingestFolder(service *drive.Service, folderId string, currentPath string) ([]*models.Document, error) {
	log.Printf("INIT: initiating folder ingestion - %s", folderId)

	var allDocuments []*models.Document
	query := fmt.Sprintf("'%s' in parents and trashed=false", folderId)
	pageToken := ""

	for {
		call := service.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name, mimeType, modifiedTime, size, parents)").
			PageSize(100)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		response, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w\n", err)
		}

		for _, file := range response.Files {
			log.Printf("INFO: examining file - %s\n", file.Name)
			filePath := filepath.Join(currentPath, file.Name)

			if file.MimeType == FOLDER_MIME_TYPE {
				subDocs, err := ingestFolder(service, file.Id, filePath)
				if err != nil {
					log.Printf("WARNING: Failed to process sub-folder '%s': %v\n", filePath, err)
				}
				allDocuments = append(allDocuments, subDocs...)
			}

			if file.MimeType == MD_MIME_TYPE || file.MimeType == TXT_MIME_TYPE {
				doc, err := extractFileContent(service, file, filePath)
				if err != nil {
					log.Printf("WARNING: Failed to extract content from '%s': %v", file.Name, err)
				}

				allDocuments = append(allDocuments, doc)
			}
		}

		pageToken = response.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return allDocuments, nil
}

func extractFileContent(service *drive.Service, file *drive.File, fullPath string) (*models.Document, error) {
	log.Printf("INFO: extracting file - %s\n", file.Name)
	response, err := service.Files.Get(file.Id).Download()
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer response.Body.Close()

	contentBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	doc := &models.Document{
		DriveFileID:  file.Id,
		FileName:     file.Name,
		FilePath:     fullPath,
		Content:      string(contentBytes),
		Extension:    strings.ToLower(filepath.Ext(file.Name)),
		LastModified: file.ModifiedTime,
		SizeBytes:    file.Size,
	}

	return doc, nil
}
