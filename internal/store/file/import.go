package file

import (
	"context"
	"fmt"

	"lowcode-wrapper/internal/importer"
)

// ImportDeclarative replaces or merges declarative metadata and persists it.
func (s *Store) ImportDeclarative(ctx context.Context, doc importer.DeclarativeDoc, mode importer.ImportMode) (importer.Result, error) {
	_ = ctx
	if mode == "" {
		mode = importer.ModeReplace
	}
	var merged importer.DeclarativeDoc
	var err error
	switch mode {
	case importer.ModeReplace:
		merged = doc
	case importer.ModeMerge:
		current, derr := decompileDeclarative(s.vault, s.data)
		if derr != nil {
			return importer.Result{}, derr
		}
		merged, err = importer.MergeDeclarativeDocs(declarativeToImporter(current), doc)
		if err != nil {
			return importer.Result{}, err
		}
	default:
		return importer.Result{}, fmt.Errorf("unsupported import mode %q", mode)
	}

	snap, err := compileDeclarative(s.vault, declarativeFromImporter(merged))
	if err != nil {
		return importer.Result{}, err
	}
	err = s.withLock(func() error {
		s.data = snap
		return nil
	})
	if err != nil {
		return importer.Result{}, err
	}
	return importer.CountResult(merged), nil
}
