package database

import (
	"time"

	"github.com/google/uuid"
)

type Resume struct {
	ID               uuid.UUID
	OriginalFilename string
	Mime             string
	SizeBytes        int64
	StorageProvider  string
	ObjectKey        string
	StorageUrl       string
	UploadStatus     string
	CreatedAt        time.Time
	SessionID        uuid.UUID
}
