package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	FOLDER_MIME_TYPE             = "application/vnd.google-apps.folder"
	FILE_MIME_TYPE               = "application/vnd.google-apps.file"
	INGESTION_PIPELINE_FOLDER_ID = "16RWlHvc-TKdqpBYDJMQdt319BS7AvjxM"
)

type Document struct {
	DriveFileID  string
	FileName     string
	FilePath     string
	Content      string
	Extension    string
	LastModified string
	SizeBytes    int64
}

func main() {
	ctx := context.Background()

	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	client := getClient(config)

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
		log.Fatalf("Failed to injest folder '%s': %v\n", folderId, err)
	}

	fmt.Printf("Ingested documents:\n")
	for _, document := range documents {
		fmt.Printf("%s (%s) - %d bytes\n", document.FilePath, document.Extension, document.SizeBytes)
	}
}

func ingestFolder(service *drive.Service, folderId string, currentPath string) ([]*Document, error) {
	log.Printf("initiating folder ingestion - %s", folderId)
	var allDocuments []*Document
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
			return nil, fmt.Errorf("Failed to list files: %w\n", err)
		}

		for _, file := range response.Files {
			filePath := filepath.Join(currentPath, file.Name)

			if file.MimeType == FOLDER_MIME_TYPE {
				subDocs, err := ingestFolder(service, folderId, filePath)
				if err != nil {
					log.Printf("Warning: Failed to process sub-folder '%s': %v\n", filePath, err)
				}
				allDocuments = append(allDocuments, subDocs...)
			}

			if file.MimeType == FILE_MIME_TYPE {
				ext := strings.ToLower(filepath.Ext(file.Name))
				if ext == ".txt" || ext == ".md" {
					doc, err := extractFileContent(service, file, filePath)
					if err != nil {
						log.Printf("Warning: Failed to extract content from '%s': %v", file.Name, err)
					}

					allDocuments = append(allDocuments, doc)
				}
			}
		}

		pageToken = response.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return allDocuments, nil
}

func extractFileContent(service *drive.Service, file *drive.File, fullPath string) (*Document, error) {
	response, err := service.Files.Get(file.Id).Download()
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer response.Body.Close()

	contentBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	doc := &Document{
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

func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	exec.Command("xdg-open", authURL).Start()

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}

	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
