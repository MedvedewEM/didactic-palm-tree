package provider

import (
	"context"
	"sort"

	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb"
	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb/models"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

type N struct {
	lg *logrus.Logger
	mdb metadb.MetaDB

	n int
}

func NewN(mdb metadb.MetaDB, n int) *N {
	return &N{
		mdb: mdb,
		n: n,
	}
}

func (p *N) Pick(ctx context.Context) ([]models.Server, error) {
	servers, err := p.mdb.ListServers(ctx, nil)
	if err != nil {
		return nil, xerrors.Errorf("list servers: %w", err)
	}

	if len(servers) < p.n {
		return nil, xerrors.Errorf("not enough servers: have: %v, should have at least: %v", len(servers), p.n)
	}

	filePartsSizes, err := p.mdb.GetFilePartsSize(ctx, nil, models.ServerToServerIDs(servers))
	if err != nil {
		return nil, xerrors.Errorf("file parts size: %w", err)
	}
	sort.Slice(servers, func(i, j int) bool {
		return filePartsSizes[servers[i].ID] < filePartsSizes[servers[j].ID]
	})

	return servers[:p.n], nil
}