package internal

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/MedvedewEM/didactic-palm-tree/internal/loader"
	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb"
	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb/models"
	"github.com/MedvedewEM/didactic-palm-tree/internal/orphaner"
	picker "github.com/MedvedewEM/didactic-palm-tree/internal/serverspicker"
	ctxio "github.com/MedvedewEM/didactic-palm-tree/pkg/ctxreader"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

type FileMetadata struct {
	Name string `json:"file_name"`
	Size int64 `json:"file_size"`
}

type App struct {
	lg *logrus.Logger
	mdb metadb.MetaDB
	picker picker.Picker
	loader loader.Loader
	orphaner chan<-orphaner.Task
}

func NewApp(
	lg *logrus.Logger,
	mdb metadb.MetaDB,
	picker picker.Picker,
	loader loader.Loader,
	orphaner chan<-orphaner.Task,
) *App {
	return &App{
		lg: lg.WithField("service", "app").Logger,
		mdb: mdb,
		picker: picker,
		loader: loader,
		orphaner: orphaner,
	}
}

func (a *App) Upload(ctx context.Context, file FileMetadata, part *multipart.Part) (uuid.UUID, error) {
	id := uuid.New()
	lg := a.lg.WithField("uuid", id.String())

	lg.Debugf("picking servers")
	pickedServers, err := a.picker.Pick(ctx)
	if err != nil {
		return uuid.UUID{}, xerrors.Errorf("pick servers: %w", err)
	}
	lg.Debugf("picked servers id: %v", models.ServersToServerIDs(pickedServers))

	var isOrphaned bool
	defer func() {
		if !isOrphaned {
			return
		}

		orphanedHosts := models.ServersToHosts(pickedServers)
		if len(orphanedHosts) == 0 {
			lg.Debugf("nothing to orphane")
			return
		}

		lg.Debugf("orphaning file in servers: %v", orphanedHosts)
		a.orphaner <- orphaner.Task{ID: id, Hosts: orphanedHosts}
	}()

	uploadedMetadata, err := a.loader.Upload(ctx, id, pickedServers, file.Size, part)
	if err != nil {
		isOrphaned = true
		return uuid.UUID{}, xerrors.Errorf("upload file parts: %w", err)
	}
	lg.Debugf("uploaded file parts, sizes: %v", uploadedMetadata)

	tx, err := a.mdb.Begin(ctx)
	if err != nil {
		isOrphaned = true
		return uuid.UUID{}, xerrors.Errorf("begin: %w", err)
	}
	defer func() {
		if err != nil {
			if err = tx.Rollback(ctx); err != nil {
				lg.Errorf("rollback tx: %v", err)
			}
		} else {
			if err := tx.Commit(ctx); err != nil {
				lg.Errorf("commit tx: %v", err)
			}
		}
	}()

	if err = a.mdb.CreateFile(ctx, tx, id, file.Name); err != nil {
		isOrphaned = true
		return uuid.UUID{}, xerrors.Errorf("create file: %w", err)
	}

	if err = a.mdb.CreateFileParts(ctx, tx, id, loader.ToPartSizes(uploadedMetadata.FileSizes)); err != nil {
		isOrphaned = true
		return uuid.UUID{}, xerrors.Errorf("create file parts: %w", err)
	}
	lg.Debugf("added file parts to MetaDB")

	return id, nil
}

func (a *App) Download(ctx context.Context, id uuid.UUID, w io.Writer) (error) {
	lg := a.lg.WithField("uuid", id.String())

	servers, err := a.mdb.ListFileServers(ctx, nil, id)
	if err != nil {
		return xerrors.Errorf("list file servers: %w", err)
	}
	lg.Debugf("file located on %v servers: %v", len(servers),  models.FilePartServersToServerIDs(servers))

	fileParts, err := a.loader.Download(ctx, id, servers)
	if err != nil {
		return xerrors.Errorf("download file parts: %w", err)
	}
	lg.Debugf("downloaded file parts")

	for i, filePart := range fileParts {
		ctxFilePart := ctxio.NewReader(ctx, filePart)
		n, err := io.Copy(w, ctxFilePart)
		if err != nil {
			return xerrors.Errorf("write file parts: %w", err)
		}
		defer filePart.Close()

		lg.Debugf("file part %v written, len: %v", i, n)
	}

	return nil
}
