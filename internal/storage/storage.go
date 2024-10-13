package storage

import (
	"io"

	"github.com/google/uuid"
)

const (
	ActionSizeInBytes = 4
	IDSizeInBytes = 16
)

type Action string

const (
	AppendAction Action = "APND"
	ReadAction Action = "READ"
	DeleteAction Action = "DLTE"
)

type Storage interface {
	Append(id uuid.UUID, data []byte) error
	Read(id uuid.UUID) (io.ReadCloser, error)
	Delete(id uuid.UUID) error
}