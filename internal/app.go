package internal

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/MedvedewEM/didactic-palm-tree/internal/loader"
	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb"
	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb/models"
	picker "github.com/MedvedewEM/didactic-palm-tree/internal/serverspicker"
	"github.com/MedvedewEM/didactic-palm-tree/internal/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

type App struct {
	lg *logrus.Logger
	mdb metadb.MetaDB

	picker picker.Picker
	loader loader.Loader
}

func NewApp(
	lg *logrus.Logger,
	mdb metadb.MetaDB,
	picker picker.Picker,
	loader loader.Loader,
) *App {
	return &App{
		lg: lg,
		mdb: mdb,
		picker: picker,
		loader: loader,
	}
}

func (a *App) Upload(filename string, file multipart.File) (uuid.UUID, error) {
	id := uuid.New()
	ctx := context.Background()
	lg := a.lg.WithField("uuid", id.String())

	lg.Debugf("picking servers")
	pickedServers, err := a.picker.Pick(ctx)
	if err != nil {
		return uuid.UUID{}, xerrors.Errorf("pick servers: %w", err)
	}
	pickedServerIDs := models.ServerToServerIDs(pickedServers)
	lg.Debugf("picked servers id: %v", pickedServerIDs)

	fileParts, err := utils.SplitByNReaders(file, int64(len(pickedServers)))
	if err != nil {
		return uuid.UUID{}, xerrors.Errorf("split readers: %w", err)
	}
	lg.Debugf("splitted by %v readers", len(fileParts))

	pickedServers = pickedServers[:len(fileParts)]
	pickedServerIDs = pickedServerIDs[:len(fileParts)]
	lg.Debugf("picked servers id after spliting file: %v", pickedServerIDs)

	uploadedPartSizes, err := a.loader.UploadFileParts(ctx, id, fileParts, pickedServers)
	if err != nil {
		return uuid.UUID{}, xerrors.Errorf("upload file parts: %w", err)
	}
	lg.Debugf("file parts uploaded, sizes: %v", uploadedPartSizes)

	tx, err := a.mdb.Begin(ctx)
	if err != nil {
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

	if err = a.mdb.CreateFile(ctx, tx, id, filename); err != nil {
		return uuid.UUID{}, xerrors.Errorf("create file: %w", err)
	}

	if err = a.mdb.CreateFileParts(ctx, tx, id, pickedServerIDs, uploadedPartSizes); err != nil {
		return uuid.UUID{}, xerrors.Errorf("create file parts: %w", err)
	}
	lg.Debugf("added file parts to MetaDB")

	return id, nil
}

func (a *App) Download(id uuid.UUID, w io.Writer) (error) {
	ctx := context.Background()
	lg := a.lg.WithField("uuid", id.String())

	servers, err := a.mdb.ListFileServers(ctx, nil, id)
	if err != nil {
		return xerrors.Errorf("list file servers: %w", err)
	}
	lg.Debugf("file located on %v servers", len(servers))

	fileParts, err := a.loader.DownloadFileParts(ctx, id, servers)
	for _, fp := range fileParts {
		defer fp.Close()
	}
	if err != nil {
		return xerrors.Errorf("download file parts: %w", err)
	}
	lg.Debugf("file parts downloaded")

	for i, filePart := range fileParts {
		n, err := io.Copy(w, filePart)
		if err != nil {
			return xerrors.Errorf("write file parts: %w", err)
		}

		lg.Debugf("file part %v written, len: %v", i, n)
	}

	return nil
}
