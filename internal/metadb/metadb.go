package metadb

import (
	"context"

	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb/models"
	"github.com/google/uuid"
)

type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type MetaDB interface {
	Begin(ctx context.Context) (Tx, error)

	ListServers(ctx context.Context, tx Tx) ([]models.Server, error)
	CreateFile(ctx context.Context, tx Tx, id uuid.UUID, filename string) error
	CreateFileParts(ctx context.Context, tx Tx, id uuid.UUID, serverIDs []int, partSizes map[int]int64) error
	ListFileServers(ctx context.Context, tx Tx, id uuid.UUID) ([]models.FilePartServer, error)
	GetFilePartsSize(ctx context.Context, tx Tx, serverIDs []int) (map[int]int64, error)
}