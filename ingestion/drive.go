package ingestion

import (
	"fmt"
	"injestion-pipeline/models"
	"log"
	"path/filepath"

	"google.golang.org/api/drive/v3"
)

type DriveIngester struct {
	service *drive.Service
}

func NewDriveIngester(service *drive.Service) *DriveIngester {
	return &DriveIngester{service: service}
}

func (d *DriveIngester) IngestFolder(folderId string, currentPath string) ([]*models.Document, error) {
	log.Printf("INIT: initiating folder ingestion - %s", folderId)

	var allDocuments []*models.Document
	query := fmt.Sprintf("'%s' in parents and trashed=false", folderId)
	pageToken := ""

	for {
		call := d.service.Files.List().
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

			if file.MimeType == FolderMimeType {
				subDocs, err := d.IngestFolder(file.Id, filePath)
				if err != nil {
					log.Printf("WARNING: Failed to process sub-folder '%s': %v\n", filePath, err)
				}
				allDocuments = append(allDocuments, subDocs...)
			}

			if file.MimeType == MarkdownMime || file.MimeType == TextMime {
				fs := NewFileProcessor(d.service)

				doc, err := fs.ExtractContent(file, filePath)
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
