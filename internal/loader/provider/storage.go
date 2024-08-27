package provider

import (
	"context"
	"io"
	"sync"

	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb/models"
	"github.com/MedvedewEM/didactic-palm-tree/pkg/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

type StorageLoader struct {
	lg *logrus.Logger
}

func NewStorageLoader(lg *logrus.Logger,) *StorageLoader {
	return &StorageLoader{
		lg: lg,
	}
}

func (l *StorageLoader) UploadFileParts(ctx context.Context, id uuid.UUID, parts []io.Reader, servers []models.Server) (map[int]int64, error) {
	eg, _ := errgroup.WithContext(ctx)
	lg := l.lg.WithField("uuid", id.String())

	partSizes := map[int]int64{}
	lock := &sync.Mutex{}
	for i, part := range parts {
		eg.Go(func() error {
			lg.Debugf("upload file part starting, part: %v", i)

			partSize, err := l.uploadFilePart(servers[i].Host, id, part)
			if err != nil {
				return xerrors.Errorf("upload file part %v: %w", i, err)
			}

			lock.Lock()
			partSizes[servers[i].ID] = partSize
			lock.Unlock()

			lg.Debugf("upload file part finished, part: %v, n: %v", i, partSize)
			return nil
		})
	}

	return partSizes, eg.Wait()
}

func (l *StorageLoader) uploadFilePart(host string, id uuid.UUID, part io.Reader) (int64, error) {
	c := storage.NewClient(host)

	return c.Upload(id, part)
}

func (l *StorageLoader) DownloadFileParts(ctx context.Context, id uuid.UUID, servers []models.FilePartServer) ([]io.ReadCloser, error) {
	eg, _ := errgroup.WithContext(ctx)
	lg := l.lg.WithField("uuid", id.String())

	fileParts := make([]io.ReadCloser, len(servers))
	for i, server := range servers {
		eg.Go(
			func() error {
				lg.Debugf("download file part starting, part: %v", i)

				filePart, err := l.downloadFilePart(server.Host, id)
				if err != nil {
					return xerrors.Errorf("download file part %v: %w", i, err)
				}

				fileParts[i] = filePart

				lg.Debugf("download file part finished, part: %v", i)
				return nil
			},
		)
	}

	return fileParts, eg.Wait()
}

func (l *StorageLoader) downloadFilePart(host string, id uuid.UUID) (io.ReadCloser, error) {
	c := storage.NewClient(host)

	return c.Download(id)
}
