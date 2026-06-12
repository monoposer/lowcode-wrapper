package postgres

import (
	"context"
	"encoding/json"

	"lowcode-wrapper/internal/importer"
)

func (s *Store) exportDeclarative(ctx context.Context) (importer.DeclarativeDoc, error) {
	servers, err := s.ListServers(ctx)
	if err != nil {
		return importer.DeclarativeDoc{}, err
	}
	tables, err := s.ListTables(ctx)
	if err != nil {
		return importer.DeclarativeDoc{}, err
	}
	functions, err := s.ListFunctions(ctx)
	if err != nil {
		return importer.DeclarativeDoc{}, err
	}

	doc := importer.DeclarativeDoc{}
	credData := map[string]map[string]any{}
	refCount := map[string]int{}

	for _, srv := range servers {
		if srv.CredentialRef != nil {
			refCount[srv.CredentialRef.String()]++
		}
	}
	for _, srv := range servers {
		enabled := srv.Enabled
		ds := importer.DeclServer{
			Name:     srv.Name,
			Protocol: string(srv.Protocol),
			Enabled:  &enabled,
		}
		if len(srv.Options) > 0 {
			_ = json.Unmarshal(srv.Options, &ds.Options)
		}
		if srv.CredentialRef != nil {
			key := srv.CredentialRef.String()
			data, ok := credData[key]
			if !ok {
				data, err = s.ResolveCredential(ctx, *srv.CredentialRef)
				if err != nil {
					return importer.DeclarativeDoc{}, err
				}
				credData[key] = data
			}
			if refCount[key] == 1 {
				ds.Credential = data
			}
		}
		doc.Servers = append(doc.Servers, ds)
	}

	for _, tbl := range tables {
		cols, err := s.ListColumns(ctx, tbl.SchemaName, tbl.TableName)
		if err != nil {
			return importer.DeclarativeDoc{}, err
		}
		dt := importer.DeclTable{
			Server:     tbl.ServerName,
			Schema:     tbl.SchemaName,
			Name:       tbl.TableName,
			RemoteName: tbl.RemoteName,
			KeyColumns: append([]string(nil), tbl.KeyColumns...),
		}
		if len(tbl.Options) > 0 {
			_ = json.Unmarshal(tbl.Options, &dt.Options)
		}
		for i, c := range cols {
			nullable := c.Nullable
			pos := c.Position
			if pos == 0 {
				pos = i
			}
			dt.Columns = append(dt.Columns, importer.DeclColumn{
				Name:       c.Name,
				DataType:   c.DataType,
				RemoteName: c.RemoteName,
				Nullable:   &nullable,
				Position:   pos,
			})
		}
		doc.Tables = append(doc.Tables, dt)
	}

	for _, fn := range functions {
		df := importer.DeclFunction{
			Server:     fn.ServerName,
			Schema:     fn.SchemaName,
			Name:       fn.Name,
			Operation:  string(fn.Operation),
			RemotePath: fn.RemotePath,
			Method:     fn.Method,
		}
		if len(fn.Options) > 0 {
			_ = json.Unmarshal(fn.Options, &df.Options)
		}
		doc.Functions = append(doc.Functions, df)
	}
	return doc, nil
}
