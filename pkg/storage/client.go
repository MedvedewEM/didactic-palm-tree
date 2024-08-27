package storage

import (
	"bytes"
	"io"
	"net"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

const (
	AppendAction = "APND"
	ReadAction = "READ"
)

type Client struct {
	host string
}

func NewClient(host string) *Client {
	return &Client{
		host: host,
	}
}

func (c *Client) Upload(id uuid.UUID, part io.Reader) (int64, error) {
	conn, err := net.Dial("tcp", c.host)
    if err != nil {
		return 0, xerrors.Errorf("net listen: %w", err)
    }
	defer conn.Close()

	uuid, _ := id.MarshalBinary()
	mr := io.MultiReader(strings.NewReader(AppendAction), bytes.NewReader(uuid), part)
	
	n, err := io.Copy(conn, mr)
	if err != nil {
		return 0, xerrors.Errorf("storage client write: %w", err)
	}

	return n-int64(len(AppendAction))-int64(len(uuid)), nil
}

func (c *Client) Download(id uuid.UUID) (io.ReadCloser, error) {
	conn, err := net.Dial("tcp", c.host)
    if err != nil {
		return nil, xerrors.Errorf("net listen: %w", err)
    }

	uuid, _ := id.MarshalBinary()
	mr := io.MultiReader(strings.NewReader(ReadAction), bytes.NewReader(uuid))

	if _, err = io.Copy(conn, mr); err != nil {
		return nil, xerrors.Errorf("net request: %w", err)
	}

	return conn, nil
}