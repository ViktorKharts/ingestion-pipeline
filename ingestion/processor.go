package ingestion

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"

	"injestion-pipeline/models"

	"google.golang.org/api/drive/v3"
)

const (
	FolderMimeType = "application/vnd.google-apps.folder"
	MarkdownMime   = "text/markdown"
	TextMime       = "text/plain"
)

type FileProcessor struct {
	service *drive.Service
}

func NewFileProcessor(service *drive.Service) *FileProcessor {
	return &FileProcessor{service: service}
}

func (p *FileProcessor) ShouldProcess(file *drive.File) bool {
	return file.MimeType == MarkdownMime || file.MimeType == TextMime
}

func (p *FileProcessor) ExtractContent(file *drive.File, fullPath string) (*models.Document, error) {
	log.Printf("INFO: extracting file - %s\n", file.Name)
	response, err := p.service.Files.Get(file.Id).Download()
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
