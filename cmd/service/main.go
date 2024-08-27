package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/MedvedewEM/didactic-palm-tree/internal"
	loader "github.com/MedvedewEM/didactic-palm-tree/internal/loader/provider"
	pg "github.com/MedvedewEM/didactic-palm-tree/internal/metadb/provider"
	picker "github.com/MedvedewEM/didactic-palm-tree/internal/serverspicker/provider"
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
    distributedStorageServers = 2
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
    connStr := fmt.Sprintf("host=%v port=%v dbname=%v user=%v", cfg.MetaDB.Host, cfg.MetaDB.Port, cfg.MetaDB.DB, cfg.MetaDB.User)
    pgPool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
        lg.Fatalf("unable to connect to database: %v", err.Error())
	}
	defer pgPool.Close()

    metadb := pg.NewPg(pgPool)

    app = internal.NewApp(
        lg,
        metadb,
        picker.NewN(metadb, cfg.App.DistributedStorageServers),
        loader.NewStorageLoader(lg),
    )

    r := http.NewServeMux()
    r.HandleFunc("/upload", uploadFile)
    r.HandleFunc("/download", downloadFile)

    srv := http.Server{
        Addr: fmt.Sprintf(":%v", cfg.Server.Port),
        Handler: r,
    }

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go func() {
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            lg.Fatalf("listen and serve: %v", err)
        }
    }()

    <-ctx.Done()
    if err := srv.Shutdown(context.TODO()); err != nil {
        lg.Printf("server shutdown: %v", err)
    }
}

func writeError(w http.ResponseWriter, wErr string) {
    w.WriteHeader(http.StatusBadRequest)
    if _, err := w.Write([]byte(wErr)); err != nil {
        panic(err)
    }
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
    r.ParseMultipartForm(cfg.Server.MaxMemoryUploadFileInBytes)

    file, header, err := r.FormFile("input")
    if err != nil {
        lg.Errorf("input form file: %v", err)
        writeError(w, errInvalidFileInput)
        return
    }
    defer file.Close()

    if header.Size > cfg.Server.MaxFileSizeInBytes {
        lg.Warningf("file size: %v", header.Size)
        writeError(w, errMaxFileSizeExceeded)
        return
    }

    id, err := app.Upload(header.Filename, file)
    if err != nil {
        lg.Errorf("upload file: %v", err)
        writeError(w, errToUploadFile)
        return
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

    if err = app.Download(uuid, w); err != nil {
        lg.Errorf("download file: %v", err)
        writeError(w, errToDownloadFile)
        return
    }
}