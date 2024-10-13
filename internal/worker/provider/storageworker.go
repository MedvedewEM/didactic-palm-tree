package provider

import (
	"bytes"
	"context"
	"time"

	"github.com/MedvedewEM/didactic-palm-tree/internal/worker"
	ctxio "github.com/MedvedewEM/didactic-palm-tree/pkg/ctxreader"
	"github.com/MedvedewEM/didactic-palm-tree/pkg/storage"
	"github.com/sirupsen/logrus"
)

type StorageWorker struct {
	ctx context.Context
	lg *logrus.Logger
	client *storage.Client
}

func NewStorageWorker(ctx context.Context, lg *logrus.Logger, client *storage.Client) worker.Worker {
	return &StorageWorker{
		ctx: ctx,
		lg: lg.WithField("service", "storage_worker").Logger,
		client: client,
	}
}

func (w *StorageWorker) Run(tasks <-chan worker.Task) {
	w.lg.Debugf("storage worker is running")

	for {
		select {
		case <-w.ctx.Done():
			w.lg.Debug("storage worker is done by ctx")
			return
		case task, ok := <-tasks:
			if !ok {
				w.lg.Debug("storage worker tasks are closed")
				return
			}

			go func(t worker.Task) {
				w.lg.Tracef("storage worker is processing %v %v", t.ID, t.StorageHost)

				ctxReader := ctxio.NewReader(w.ctx, bytes.NewReader(t.Buf))
	
				n, err := w.client.Upload(t.StorageHost, t.ID, ctxReader)
	
				t.Result <- worker.Result{n, err}
	
				w.lg.Tracef("storage worker is processed %v %v", t.ID, t.StorageHost)
			}(task)
		}
	}
}

func (o *StorageWorker) Shutdown(ctx context.Context) {
	shutdownCtx, _ := context.WithTimeout(ctx, 5 * time.Second)

	o.ctx = shutdownCtx
}
