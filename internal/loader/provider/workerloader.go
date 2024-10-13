package provider

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/MedvedewEM/didactic-palm-tree/internal"
	"github.com/MedvedewEM/didactic-palm-tree/internal/loader"
	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb/models"
	"github.com/MedvedewEM/didactic-palm-tree/internal/worker"
	"github.com/MedvedewEM/didactic-palm-tree/pkg/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

type WorkerLoader struct {
	lg *logrus.Logger
	cfg internal.LoaderConfig
	client *storage.Client
	worker chan<- worker.Task
}

func NewWorkerLoader(lg *logrus.Logger, cfg internal.LoaderConfig, client *storage.Client, worker chan<- worker.Task) loader.Loader {
	return &WorkerLoader{
		lg: lg.WithField("service", "worker_loader").Logger,
		cfg: cfg,
		client: client,
		worker: worker,
	}
}

func (l *WorkerLoader) Upload(ctx context.Context, id uuid.UUID, servers []models.Server, fileSize int64, part *multipart.Part) (loader.UploadedMetadata, error) {
	lg := l.lg.WithField("uuid", id.String())

	if len(servers) == 0 {
		return loader.UploadedMetadata{}, xerrors.Errorf("invalid servers")
	}
	if part == nil {
		return loader.UploadedMetadata{}, xerrors.Errorf("invalid multipart")
	}
	if fileSize < 0 {
		return loader.UploadedMetadata{}, xerrors.Errorf("invalid file size: %v", fileSize)
	}

	if fileSize == 0 {
		return loader.UploadedMetadata{FileSizes: nil}, nil
	}

	bufBytes := l.cfg.UploadFileMaxBufferInBytes
	if fileSize > int64(len(servers)) * l.cfg.UploadLargeFileMaxBufferInBytes {
		bufBytes = l.cfg.UploadLargeFileMaxBufferInBytes
	}

	lg.Tracef("uploading size %v to servers %v by buf size %v", fileSize, servers, bufBytes)

	metadata := loader.UploadedMetadata{
		FileSizes: map[loader.OrderedServer]int64{},
	}

	for _, server := range servers {
		defer l.client.Reset(server.Host, id)
	}

	uploaded, iServer := int64(0), 0
	nth := fileSize / int64(len(servers))
	for {
		select {
		case <-ctx.Done():
			close(l.worker)

			var errStr string
			switch ctx.Err() {
			case context.Canceled:
				errStr = "ctx is canceled"
			case context.DeadlineExceeded:
				errStr = "ctx is deadline exceeded"
			default:
				errStr = "ctx is done"
			}
			lg.Debugf(errStr)
			return loader.UploadedMetadata{}, xerrors.Errorf(errStr)
		default:
			buf := make([]byte, bufBytes)
			n, err := part.Read(buf)
			if err != nil && err != io.EOF {
				return loader.UploadedMetadata{}, xerrors.Errorf("part read: %w", err)
			}

			if n > 0 {
				buf = buf[:n]

				lg.Tracef("uploading bytes %v to server %v", len(buf), servers[iServer].Host)

				result := make(chan worker.Result)
				l.worker <- worker.Task{
					ID: id,
					StorageHost: servers[iServer].Host,
					Buf: buf,
					Result: result,
				}
				res := <-result

				if res.Err != nil {
					return loader.UploadedMetadata{}, xerrors.Errorf("task failed: %w", res.Err)
				}
				if int64(n) != res.Len {
					return loader.UploadedMetadata{}, xerrors.Errorf("inconsistent task upload: %v != %v", int64(n), res.Len)
				}

				uploaded += res.Len
				if uploaded > fileSize {
					return loader.UploadedMetadata{}, xerrors.Errorf("uploaded more than expected: %v != %v", int64(fileSize), uploaded)
				}

				metadata.FileSizes[loader.OrderedServer{servers[iServer].ID, iServer}] += res.Len

				lg.Tracef("uploaded total %v bytes", uploaded)
			}

			if uploaded >= nth*(int64(iServer) + 1) && iServer != len(servers)-1 {
				lg.Tracef("pick next server: %v", servers[iServer].Host)
				iServer++
			}

			if err == io.EOF {
				if uploaded < fileSize {
					return loader.UploadedMetadata{}, xerrors.Errorf("uploaded less than expected: %v != %v", int64(fileSize), uploaded)
				}

				lg.Tracef("uploaded metadata: %v", metadata)
				return metadata, nil
			}
		}
	}
}

func (l *WorkerLoader) Download(ctx context.Context, id uuid.UUID, servers []models.FilePartServer) ([]io.ReadCloser, error) {
	eg, _ := errgroup.WithContext(ctx)
	lg := l.lg.WithField("uuid", id.String())

	fileParts := make([]io.ReadCloser, len(servers))
	for i, server := range servers {
		eg.Go(
			func() error {
				lg.Debugf("downloading file part: %v", i)

				filePart, err := l.downloadFilePart(server.Host, id)
				if err != nil {
					return xerrors.Errorf("download file part %v: %w", i, err)
				}

				fileParts[i] = filePart

				lg.Debugf("downloaded file part: %v", i)
				return nil
			},
		)
	}

	return fileParts, eg.Wait()
}

func (l *WorkerLoader) downloadFilePart(host string, id uuid.UUID) (io.ReadCloser, error) {
	return l.client.Download(host, id)
}
