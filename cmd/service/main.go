package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MedvedewEM/didactic-palm-tree/internal"
	loaderProv "github.com/MedvedewEM/didactic-palm-tree/internal/loader/provider"
	pgProv "github.com/MedvedewEM/didactic-palm-tree/internal/metadb/provider"
	"github.com/MedvedewEM/didactic-palm-tree/internal/orphaner"
	orphanerProv "github.com/MedvedewEM/didactic-palm-tree/internal/orphaner/provider"
	pickerProv "github.com/MedvedewEM/didactic-palm-tree/internal/serverspicker/provider"
	"github.com/MedvedewEM/didactic-palm-tree/internal/worker"
	workerProv "github.com/MedvedewEM/didactic-palm-tree/internal/worker/provider"
	"github.com/MedvedewEM/didactic-palm-tree/pkg/storage"
	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

const (
    errInvalidFileInput = "Invalid file input."
    errMaxFileSizeExceeded = "Max file size exceeded."
    errToUploadFile = "Error to upload file. Please, try again later."
    errInvalidUUID = "Invalid UUID input."
    errToDownloadFile = "Error to download file. Please, try again later."
)

const (
    configPath = "config.yml"
)

var cfg internal.Config
var lg *logrus.Logger
var app *internal.App

func init() {
    err := cleanenv.ReadConfig(configPath, &cfg)
    if err != nil {
       panic(fmt.Sprintf("read config: %q, err: %v", configPath, err))
    }

    cfg = internal.MergeConfigSections(internal.NewDefaultConfig(), cfg)

    initLogger(*cfg.Logger)

    lg.Printf("Configuration: %+v %+v %+v %+v %+v", cfg, *cfg.App, *cfg.Logger, *cfg.MetaDB, *cfg.Server)
}

func initLogger(cfg internal.LoggerConfig) {
    var output io.Writer
    switch cfg.Output {
    case "stdout":
        output = os.Stdout
    default:
        output = os.Stdout
    }

    lg = logrus.New()
    lg.SetLevel(logrus.Level(cfg.Level))
    lg.SetOutput(output)
}

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    connStr := fmt.Sprintf("host=%v port=%v dbname=%v user=%v", cfg.MetaDB.Host, cfg.MetaDB.Port, cfg.MetaDB.DB, cfg.MetaDB.User)
    pgPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
        lg.Fatalf("unable to connect to database: %v", err.Error())
	}
	defer pgPool.Close()

    metadb := pgProv.NewPg(pgPool)

    workerTasks := make(chan worker.Task)
    orphanerTasks := make(chan orphaner.Task)

    app = internal.NewApp(
        lg,
        metadb,
        pickerProv.NewN(metadb, cfg.App.DistributedStorageServers),
        loaderProv.NewWorkerLoader(lg, *cfg.Loader, storage.NewClient(), workerTasks),
        orphanerTasks,
    )

    workerCtx := context.Background()
    worker := workerProv.NewStorageWorker(workerCtx, lg, storage.NewClient())
    go worker.Run(workerTasks)

    orphanerCtx := context.Background()
    orphaner := orphanerProv.NewStorageOrphaner(orphanerCtx, lg, storage.NewClient())
    go orphaner.Run(orphanerTasks)

    r := http.NewServeMux()
    r.HandleFunc("/upload", uploadFile)
    r.HandleFunc("/download", downloadFile)

    srv := http.Server{
        Addr: fmt.Sprintf(":%v", cfg.Server.Port),
        Handler: r,
    }

    go func() {
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            lg.Fatalf("listen and serve: %v", err)
        }
    }()

    <-ctx.Done()

    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10 * time.Second)
    defer shutdownCancel()

    if err := srv.Shutdown(shutdownCtx); err != nil {
        lg.Errorf("server shutdown: %v", err)
    }

    worker.Shutdown(workerCtx)
    orphaner.Shutdown(orphanerCtx)
}

func writeError(w http.ResponseWriter, wErr string) {
    w.WriteHeader(http.StatusBadRequest)
    if _, err := w.Write([]byte(wErr)); err != nil {
        panic(err)
    }
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, cfg.Server.UploadMaxFileSizeInBytes)

	multipartReader, err := r.MultipartReader()
	if err != nil {
        lg.Errorf("multipart reader: %v", err)
        writeError(w, errToUploadFile)
		return
	}

    ctx := r.Context()

    processed := map[string]struct{}{}
    metadata := &internal.FileMetadata{}
    var id uuid.UUID
    for part, partErr := multipartReader.NextPart(); partErr == nil; part, partErr = multipartReader.NextPart() {
        formName := part.FormName()
        if _, ok := processed[formName]; ok {
            lg.Warnf("%q already processed", formName)
            break
        }
        processed[formName] = struct{}{}

        switch formName {
        case "metadata":
            buf := make([]byte, cfg.Server.UploadMaxMetadataInBytes)
            nRead := 0
            for {
                n, err := part.Read(buf[nRead:])
                nRead += n
                if err != nil || n == 0 {
                    if err == io.EOF {
                        break
                    }

                    lg.Errorf("metadata read: %v", err)
                    writeError(w, errToUploadFile)
                    return
                }
            }

            buf = buf[:nRead]

            if err := json.Unmarshal(buf, metadata); err != nil {
                lg.Errorf("metadata unmarshal: %v %v", err, string(buf))
                writeError(w, errToUploadFile)
                return
            }
            
        case "input":
            if metadata.Name == "" || metadata.Size < 0 {
                lg.Errorf("empty input parameters")
                writeError(w, errToUploadFile)
                return
            }
        
            id, err = app.Upload(ctx, *metadata, part)
            if err != nil {
                lg.Errorf("upload file: %v", err)
                writeError(w, errToUploadFile)
                return
            }
            err = part.Close()
            if err != nil {
                lg.Errorf("filepart close: %v", err)
            }
        }
    }

    if _, err := w.Write([]byte(id.String())); err != nil {
        lg.Errorf("write response: %v", err)
        writeError(w, errToUploadFile)
        return
    }
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("uuid")
    uuid, err := uuid.Parse(id)
    if err != nil {
        lg.Errorf("parse uuid: %v", err)
        writeError(w, errInvalidUUID)
        return
    }

    ctx := r.Context()

    if err = app.Download(ctx, uuid, w); err != nil {
        lg.Errorf("download file: %v", err)
        writeError(w, errToDownloadFile)
        return
    }
}