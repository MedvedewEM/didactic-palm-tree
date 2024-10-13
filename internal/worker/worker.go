package worker

import (
	"context"

	"github.com/google/uuid"
)

type Worker interface {
	Run(tasks <-chan Task)
	Shutdown(ctx context.Context)
}

type Task struct {
	ID uuid.UUID
	StorageHost string
	Buf []byte
	Result chan<- Result
}

type Result struct {
	Len int64
	Err error
}
