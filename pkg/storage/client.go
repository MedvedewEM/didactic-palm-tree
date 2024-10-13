package storage

import (
	"bytes"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

const (
	AppendAction = "APND"
	ReadAction = "READ"
	DeleteAction = "DLTE"
)

type connPoolKey struct {
	id uuid.UUID
	host string
}

type Client struct {
	uploadConnPool map[connPoolKey]net.Conn
}

func NewClient() *Client {
	return &Client{
		uploadConnPool: map[connPoolKey]net.Conn{},
	}
}

func (c *Client) Upload(host string, id uuid.UUID, part io.Reader) (int64, error) {
	var ok bool
	var conn net.Conn
	var err error
	var headerLen int

	uuid, _ := id.MarshalBinary()

	poolKey := connPoolKey{id, host}
	reader := part
	if conn, ok = c.uploadConnPool[poolKey]; !ok {
		conn, err = net.Dial("tcp", host)
		if err != nil {
			return 0, xerrors.Errorf("net dial: %w", err)
		}

		c.uploadConnPool[poolKey] = conn
		
		headerReader := io.MultiReader(strings.NewReader(AppendAction), bytes.NewReader(uuid))
		reader = io.MultiReader(headerReader, part)

		headerLen = len(AppendAction)+len(uuid)
	}
	
	n, err := io.Copy(conn, reader)
	if err != nil {
		return 0, xerrors.Errorf("net upload request: %w", err)
	}

	return n-int64(headerLen), nil
}

func (c *Client) Download(host string, id uuid.UUID) (io.ReadCloser, error) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, xerrors.Errorf("net dial: %w", err)
	}

	uuid, _ := id.MarshalBinary()
	mr := io.MultiReader(strings.NewReader(ReadAction), bytes.NewReader(uuid))

	if _, err = io.Copy(conn, mr); err != nil {
		return nil, xerrors.Errorf("net download request: %w", err)
	}

	return conn, nil
}

func (c *Client) Delete(id uuid.UUID, hosts []string) error {
	wg := sync.WaitGroup{}
	errs := make(chan error)
	defer close(errs)

	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			err := c.delete(id, host)
			if err != nil {
				errs <- err
			}

			wg.Done()
		}(host)
	}

	wg.Wait()

	allErrs := []string{}
	for err := range errs {
		allErrs = append(allErrs, "'", err.Error(), "'")
	}
	
	if len(allErrs) > 0 {
		return xerrors.Errorf("delete errors: (%v)", strings.Join(allErrs, " AND "))
	}

	return nil
}

func (c *Client) delete(id uuid.UUID, host string) error {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return xerrors.Errorf("net dial: %w", err)
	}

	uuid, _ := id.MarshalBinary()
	mr := io.MultiReader(strings.NewReader(DeleteAction), bytes.NewReader(uuid))

	if _, err = io.Copy(conn, mr); err != nil {
		return xerrors.Errorf("net delete request: %w", err)
	}

	return nil
}

func (c *Client) Reset(host string, id uuid.UUID) error {
	poolKey := connPoolKey{id, host}
	if conn, ok := c.uploadConnPool[poolKey]; ok {
		err := conn.Close()
		if err != nil {
			return err
		}

		delete(c.uploadConnPool, poolKey)
	}
	return nil
}