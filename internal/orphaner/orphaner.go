package orphaner

import (
	"context"

	"github.com/google/uuid"
)

type Orphaner interface {
	Run(tasks <-chan Task)
	Shutdown(ctx context.Context)
}

type Task struct {
	ID uuid.UUID
	Hosts []string
}
