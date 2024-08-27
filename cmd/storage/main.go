package main

import (
	"io"
	"log"
	"net"
	"os"

	"github.com/MedvedewEM/didactic-palm-tree/internal/storage"
	storageProv "github.com/MedvedewEM/didactic-palm-tree/internal/storage/provider"
	"github.com/alecthomas/units"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

var fileStorage = storageProv.NewDisk()

func handleRead(lg *logrus.Entry, conn net.Conn, id uuid.UUID) error {
	defer func() {
		conn.Close()
		lg.Println("connection closed")
	}()

	lg.Debugf("reading from file storage")
	f, err := fileStorage.Read(id)
	if err != nil {
		return xerrors.Errorf("can not read data: %w", err)
	}
	defer f.Close()

	lg.Debugf("coping to connection")
	n, err := io.Copy(conn, f)
	if err != nil {
		return xerrors.Errorf("can not write data: %w", err)
	}

	lg.Printf("copied, bytes len: %v", n)

	return nil
}

func handleAppend(lg *logrus.Entry, conn net.Conn, id uuid.UUID) error {
	defer func() {
		conn.Close()
		lg.Println("connection closed")
	}()

	buf := make([]byte, 100 * units.MB)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			lg.Printf("reading buffer, bytes len: %v", n)
			fileStorage.Append(id, buf[:n])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return xerrors.Errorf("can not read data: %w", err)
		}
	}

	return nil
}

func handle(conn net.Conn) error {
	defaultLogger := newLog()

	actionBuf := make([]byte, storage.ActionSizeInBytes)
	if _, err := conn.Read(actionBuf); err != nil {
		return xerrors.Errorf("read action: %w", err)
	}
	action := storage.Action(actionBuf)

	idBuf := make([]byte, storage.IDSizeInBytes)
	if _, err := conn.Read(idBuf); err != nil {
		return xerrors.Errorf("read id: %w", err)
	}
	uuid, err := uuid.FromBytes(idBuf)
	if err != nil {
		xerrors.Errorf("parse uuid: %w", err)
	}

	lg := defaultLogger.WithFields(logrus.Fields{"action": action, "uuid": uuid.String()})

	switch action {
	case storage.ReadAction:
		return handleRead(lg, conn, uuid)
	case storage.AppendAction:
		return handleAppend(lg, conn, uuid)
	default:
		return xerrors.Errorf("unknown action: %v", string(actionBuf))
	}
}

func newLog() *logrus.Logger {
    lg := logrus.New()
    lg.SetLevel(logrus.TraceLevel)
    lg.SetOutput(os.Stdout)

	return lg
}

func main() {
	server, _ := net.Listen("tcp", ":8081")
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("accept: %w", err)
			return
		}

		log.Println("connection accepted")

		go func() {
			log.Println("connection run")
			if err := handle(conn); err != nil {
				log.Fatal(err)
			}
		}()
	}
}