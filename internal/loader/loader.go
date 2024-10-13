package loader

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb"
	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb/models"
	"github.com/google/uuid"
)

type Loader interface {
	Upload(ctx context.Context, id uuid.UUID, servers []models.Server, fileSize int64, part *multipart.Part) (UploadedMetadata, error)
	Download(ctx context.Context, id uuid.UUID, servers []models.FilePartServer) ([]io.ReadCloser, error)
}

type UploadedMetadata struct {
	FileSizes map[OrderedServer]int64
}

type OrderedServer struct {
	ServerID int
	Order int
}

func ToPartSizes(fileSizes map[OrderedServer]int64) map[metadb.PartSizeKey]int64 {
	ps := map[metadb.PartSizeKey]int64{}
	for k, v := range fileSizes {
		ps[metadb.PartSizeKey{ServerID: k.ServerID, Order: k.Order}] = v
	}

	return ps
}
