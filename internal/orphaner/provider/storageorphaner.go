package provider

import (
	"context"
	"time"

	"github.com/MedvedewEM/didactic-palm-tree/internal/orphaner"
	"github.com/MedvedewEM/didactic-palm-tree/pkg/storage"
	"github.com/sirupsen/logrus"
)

type StorageOrphaner struct {
	ctx context.Context
	lg *logrus.Logger
	client *storage.Client
}

func NewStorageOrphaner(ctx context.Context, lg *logrus.Logger, client *storage.Client) orphaner.Orphaner {
	return &StorageOrphaner{
		ctx: ctx,
		lg: lg.WithField("service", "storage_orphaner").Logger,
		client: client,
	}
}

func(o *StorageOrphaner) Run(tasks <-chan orphaner.Task) {
	o.lg.Debugf("storage orphaner running")

	for {
		select {
		case <-o.ctx.Done():
			o.lg.Debug("storage orphaner is done by ctx")
			return
		case task, ok := <-tasks:
			if !ok {
				o.lg.Debug("storage orphaner tasks are closed")
				return
			}

			go func(t orphaner.Task) {
				o.lg.Tracef("storage orphaner is processing %v", t.ID.String())

				err := o.client.Delete(t.ID, t.Hosts)
				if err != nil {
					o.lg.Errorf("storage orphaner task processing error: %v", err)
					return
				}
	
				o.lg.Tracef("storage orphaner is processed %v", t.ID.String())
			}(task)
		}
	}
}

func (o *StorageOrphaner) Shutdown(ctx context.Context) {
	shutdownCtx, _ := context.WithTimeout(ctx, 5 * time.Second)

	o.ctx = shutdownCtx
}
