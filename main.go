package main

import (
	"context"
	"fmt"
	"injestion-pipeline/auth"
	"injestion-pipeline/models"
	"injestion-pipeline/storage"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	FOLDER_MIME_TYPE             = "application/vnd.google-apps.folder"
	MD_MIME_TYPE                 = "text/markdown"
	TXT_MIME_TYPE                = "text/plain"
	INGESTION_PIPELINE_FOLDER_ID = "16RWlHvc-TKdqpBYDJMQdt319BS7AvjxM"
	DEFAULT_DB_PATH              = "knowledge.db"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "search" {
		runSearch()
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "list" {
		runList()
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "clear" {
		runClear()
		return
	}

	runIngest()
}

func runIngest() {
	ctx := context.Background()

	db := storage.NewSQLiteDB(DEFAULT_DB_PATH)
	if err := db.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	config := auth.Config{
		CredentialsPath: "credentials.json",
		TokenPath:       "token.json",
		Scopes:          []string{drive.DriveReadonlyScope},
	}

	authenticator, err := auth.NewGoogleAuthenticator(config)
	if err != nil {
		log.Fatalf("Failed to instantiate authenticator: %v", err)
	}

	client, err := authenticator.GetHTTPClient(ctx)
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}

	service, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	folderId := INGESTION_PIPELINE_FOLDER_ID
	if len(os.Args) > 1 {
		folderId = os.Args[1]
	}

	documents, err := ingestFolder(service, folderId, "/")
	if err != nil {
		log.Fatalf("Failed to ingest folder '%s': %v\n", folderId, err)
	}

	log.Printf("INFO: Saving %d documents to database...\n", len(documents))
	saved := 0
	for _, document := range documents {
		err := db.SaveDocument(ctx, document)
		if err != nil {
			log.Printf("Warning: Failed to save document %s: %v\n", document.FileName, err)
		} else {
			saved++
			log.Printf("✓ Saved: %s\n", document.FilePath)
		}
	}

	log.Printf("Ingestion complete! Saved %d/%d documents to database.\n", saved, len(documents))
	log.Printf("Use './pipeline search <query>' to search the knowledge base\n")
}

func runSearch() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:		./pipeline search <query>")
		fmt.Println("Example:	./pipeline search \"login\"")
		os.Exit(1)
	}

	query := strings.Join(os.Args[2:], " ")
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
		return
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

func runList() {
	ctx := context.Background()

	db := storage.NewSQLiteDB(DEFAULT_DB_PATH)
	if err := db.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	docs, err := db.ListAllDocuments(ctx)
	if err != nil {
		log.Fatalf("Failed to list documents: %v", err)
	}

	fmt.Printf("INFO: Total documents in database: %d\n\n", len(docs))

	for i, doc := range docs {
		fmt.Printf("%d. %s (%s) - %d bytes\n", i+1, doc.Filepath, doc.Extension, doc.SizeBytes)
	}
}

func runClear() {
	ctx := context.Background()

	db := storage.NewSQLiteDB(DEFAULT_DB_PATH)
	if err := db.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	fmt.Print("WARNING: Are you sure you want to clear all documents? (yes/no): ")
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "yes" {
		log.Printf("INFO: Cancelled.\n")
		return
	}

	err := db.ClearAll(ctx)
	if err != nil {
		log.Fatalf("Failed to clear database: %v", err)
	}

	log.Printf("INFO: All documents cleared from database.\n")
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
