package store

import (
	"context"
	"fmt"

	"github.com/monoposer/dataspan/internal/importer"
	"github.com/monoposer/dataspan/internal/store/file"
	"github.com/monoposer/dataspan/internal/store/postgres"
)

type Importer interface {
	ImportDeclarative(ctx context.Context, doc importer.DeclarativeDoc, mode importer.ImportMode) (importer.Result, error)
}

// ImportDeclarative applies declarative metadata to the active store backend.
func ImportDeclarative(ctx context.Context, s Store, doc importer.DeclarativeDoc, mode importer.ImportMode) (importer.Result, error) {
	switch st := s.(type) {
	case *file.Store:
		return st.ImportDeclarative(ctx, doc, mode)
	case *postgres.Store:
		return st.ImportDeclarative(ctx, doc, mode)
	default:
		return importer.Result{}, fmt.Errorf("import not supported for this store backend")
	}
}

// ImportYAML parses declarative YAML and imports it into the store.
func ImportYAML(ctx context.Context, s Store, raw []byte, mode importer.ImportMode) (importer.Result, error) {
	doc, err := importer.ParseDeclarativeYAML(raw)
	if err != nil {
		return importer.Result{}, err
	}
	return ImportDeclarative(ctx, s, doc, mode)
}
