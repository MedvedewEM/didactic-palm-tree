package provider

import (
	"io"
	"os"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

type Disk struct {
}

func NewDisk() *Disk {
	return &Disk{}
}

func (d *Disk) Append(id uuid.UUID, data []byte) error {
	f, err := os.OpenFile(id.String(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return xerrors.Errorf("open file: %w", err)
	}
	defer f.Close()

	n, err := f.Write(data)
	if err != nil {
		return xerrors.Errorf("write file: %w", err)
	}

	if n != len(data) {
		return xerrors.New("inconsistency while writing")
	}

	return nil
}

func (d *Disk) Read(id uuid.UUID) (io.ReadCloser, error) {
	f, err := os.Open(id.String())
	if err != nil {
		return nil, xerrors.Errorf("open file: %w", err)
	}

	return f, nil
}
