package pg

import (
	"context"

	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb"
	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/xerrors"
)

const (
	sqlListServers = `
		SELECT id, host
		FROM servers
		ORDER BY id
	`

	sqlCreateFile = `
		INSERT INTO files (id, name)
		VALUES (@id, @name)
	`

	sqlCreateFileParts = `
		INSERT INTO file_parts (file_id, server_id, part_num, part_size)
		VALUES (@file_id, @server_id, @part_num, @part_size)
	`

	sqlListFileServers = `
		SELECT
			s.id,
			s.host,
			fp.part_size
		FROM file_parts fp
		LEFT JOIN servers s ON fp.server_id = s.id
		WHERE fp.file_id = @file_id
		ORDER BY fp.part_num
	`

	sqlGetFilePartsSize = `
		SELECT s.id AS server_id, COALESCE(SUM(fp.part_size), 0) AS parts_size
		FROM servers s
		JOIN file_parts fp ON s.id = fp.server_id
		WHERE s.id = any (@server_ids)
		GROUP BY s.id
		ORDER BY SUM(fp.part_size)
	`
)

type PgTx struct {
	tx pgx.Tx
}

func (p *PgTx) Commit(ctx context.Context) error {
	return p.tx.Commit(ctx)
}

func (p *PgTx) Rollback(ctx context.Context) error {
	return p.tx.Rollback(ctx)
}

type conn interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

type Pg struct {
	pool *pgxpool.Pool
}

func NewPg(pgPool *pgxpool.Pool) *Pg {
	return &Pg{
		pool: pgPool,
	}
}

func (pg *Pg) Begin(ctx context.Context) (metadb.Tx, error) {
	tx, err := pg.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &PgTx{tx: tx}, nil
}

func (pg *Pg) conn(tx metadb.Tx) conn {
	pgtx, ok := tx.(*PgTx)
	if !ok || pgtx == nil {
		return pg.pool
	}

	return pgtx.tx
}

func (pg *Pg) ListServers(ctx context.Context, tx metadb.Tx) ([]models.Server, error) {
	rows, err := pg.conn(tx).Query(ctx, sqlListServers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[models.Server])
}

func (pg *Pg) CreateFile(ctx context.Context, tx metadb.Tx, id uuid.UUID, filename string) error {
	_, err := pg.conn(tx).Exec(ctx, sqlCreateFile, pgx.NamedArgs{"id": id.String(), "name": filename})
	if err != nil {
		return err
	}

	return nil
}

func (pg *Pg) CreateFileParts(ctx context.Context, tx metadb.Tx, id uuid.UUID, serverIDs []int, partSizes map[int]int64) error {
	if len(serverIDs) == 0 {
		return nil
	}

	batches := &pgx.Batch{}
	for i, serverID := range serverIDs {
		partSize, ok := partSizes[serverID]
		if !ok {
			return xerrors.Errorf("create file part for server_id: %v", serverID)
		}

		batches.Queue(sqlCreateFileParts, pgx.NamedArgs{
			"file_id": id.String(),
			"server_id": serverID,
			"part_num": i,
			"part_size": partSize,
		})
	}

	results := pg.conn(tx).SendBatch(ctx, batches)
	defer results.Close()

    for range serverIDs {
		if _, err := results.Exec(); err != nil {
			return err
		}
    }

	return nil
}

func (pg *Pg) ListFileServers(ctx context.Context, tx metadb.Tx, id uuid.UUID) ([]models.FilePartServer, error) {
	rows, err := pg.conn(tx).Query(ctx, sqlListFileServers, pgx.NamedArgs{"file_id": id.String()})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[models.FilePartServer])
}

func (pg *Pg) GetFilePartsSize(ctx context.Context, tx metadb.Tx, serverIDs []int) (map[int]int64, error) {
	rows, err := pg.conn(tx).Query(ctx, sqlGetFilePartsSize, pgx.NamedArgs{"server_ids": serverIDs})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type filePartsSize struct {
		ServerID int
		PartsSize int64
	}

	filePartsSizes, err := pgx.CollectRows(rows, pgx.RowToStructByName[filePartsSize])
	if err != nil {
		return nil, err
	}

	result := map[int]int64{}
	for _, fps := range filePartsSizes {
		result[fps.ServerID] = fps.PartsSize
	}

	return result, nil
}
