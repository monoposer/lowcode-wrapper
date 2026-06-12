package file

import (
	"lowcode-wrapper/internal/auth"
	"lowcode-wrapper/internal/importer"
)

func declarativeFromImporter(doc importer.DeclarativeDoc) declarativeDoc {
	out := declarativeDoc{
		Servers:   make([]declServer, 0, len(doc.Servers)),
		Tables:    make([]declTable, 0, len(doc.Tables)),
		Functions: make([]declFunction, 0, len(doc.Functions)),
	}
	for _, c := range doc.Credentials {
		out.Credentials = append(out.Credentials, declCredential{
			Name: c.Name,
			Data: c.Data,
		})
	}
	for _, s := range doc.Servers {
		ds := declServer{
			Name:     s.Name,
			Protocol: s.Protocol,
			Enabled:  s.Enabled,
			Options:  s.Options,
		}
		if len(s.Credential) > 0 {
			ds.Credential = declCredentialField{Data: s.Credential}
		}
		out.Servers = append(out.Servers, ds)
	}
	for _, t := range doc.Tables {
		cols := make([]declColumn, len(t.Columns))
		for i, c := range t.Columns {
			cols[i] = declColumn{
				Name:       c.Name,
				DataType:   c.DataType,
				RemoteName: c.RemoteName,
				Nullable:   c.Nullable,
				Position:   c.Position,
			}
		}
		out.Tables = append(out.Tables, declTable{
			Server:     t.Server,
			Schema:     t.Schema,
			Name:       t.Name,
			RemoteName: t.RemoteName,
			KeyColumns: t.KeyColumns,
			Options:    t.Options,
			Columns:    cols,
		})
	}
	for _, f := range doc.Functions {
		out.Functions = append(out.Functions, declFunction{
			Server:     f.Server,
			Schema:     f.Schema,
			Name:       f.Name,
			Operation:  f.Operation,
			RemotePath: f.RemotePath,
			Method:     f.Method,
			Options:    f.Options,
		})
	}
	return out
}

func declarativeToImporter(doc declarativeDoc) importer.DeclarativeDoc {
	out := importer.DeclarativeDoc{
		Servers:   make([]importer.DeclServer, 0, len(doc.Servers)),
		Tables:    make([]importer.DeclTable, 0, len(doc.Tables)),
		Functions: make([]importer.DeclFunction, 0, len(doc.Functions)),
	}
	for _, c := range doc.Credentials {
		out.Credentials = append(out.Credentials, importer.DeclCredential{
			Name: c.Name,
			Data: c.Data,
		})
	}
	for _, s := range doc.Servers {
		ds := importer.DeclServer{
			Name:     s.Name,
			Protocol: s.Protocol,
			Enabled:  s.Enabled,
			Options:  s.Options,
		}
		if len(s.Credential.Data) > 0 {
			ds.Credential = s.Credential.Data
		}
		out.Servers = append(out.Servers, ds)
	}
	for _, t := range doc.Tables {
		cols := make([]importer.DeclColumn, len(t.Columns))
		for i, c := range t.Columns {
			cols[i] = importer.DeclColumn{
				Name:       c.Name,
				DataType:   c.DataType,
				RemoteName: c.RemoteName,
				Nullable:   c.Nullable,
				Position:   c.Position,
			}
		}
		out.Tables = append(out.Tables, importer.DeclTable{
			Server:     t.Server,
			Schema:     t.Schema,
			Name:       t.Name,
			RemoteName: t.RemoteName,
			KeyColumns: t.KeyColumns,
			Options:    t.Options,
			Columns:    cols,
		})
	}
	for _, f := range doc.Functions {
		out.Functions = append(out.Functions, importer.DeclFunction{
			Server:     f.Server,
			Schema:     f.Schema,
			Name:       f.Name,
			Operation:  f.Operation,
			RemotePath: f.RemotePath,
			Method:     f.Method,
			Options:    f.Options,
		})
	}
	return out
}

// CompileImporterDoc normalizes an importer declarative document into an in-memory snapshot.
func CompileImporterDoc(vault *auth.Vault, doc importer.DeclarativeDoc) (snapshot, error) {
	return compileDeclarative(vault, declarativeFromImporter(doc))
}
