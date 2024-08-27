package loader

import (
	"context"
	"io"

	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb/models"
	"github.com/google/uuid"
)

type Loader interface {
	UploadFileParts(ctx context.Context, id uuid.UUID, parts []io.Reader, servers []models.Server) (map[int]int64, error)
	DownloadFileParts(ctx context.Context, id uuid.UUID, servers []models.FilePartServer) ([]io.ReadCloser, error)
}