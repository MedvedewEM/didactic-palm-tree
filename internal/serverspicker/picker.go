package picker

import (
	"context"

	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb/models"
)

type Picker interface {
	Pick(ctx context.Context) ([]models.Server, error)
}