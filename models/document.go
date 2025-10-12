package models

type Document struct {
	DriveFileID  string
	FileName     string
	FilePath     string
	Content      string
	Extension    string
	LastModified string
	SizeBytes    int64
}
