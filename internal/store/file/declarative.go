package file

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"lowcode-wrapper/internal/auth"
)

// declarativeDoc is the file-mode drivers.yaml shape. It intentionally differs from
// postgres Meta DB tables (no top-level credentials list required): secrets sit inline
// on servers. compileDeclarative normalizes into the same in-memory snapshot as postgres.
type declarativeDoc struct {
	Credentials []declCredential `yaml:"credentials,omitempty"` // legacy / shared creds only
	Servers     []declServer     `yaml:"servers"`
	Tables      []declTable      `yaml:"tables"`
	Functions   []declFunction   `yaml:"functions"`
}

type declCredential struct {
	Name string         `yaml:"name"`
	Data map[string]any `yaml:"data"`
}

type declServer struct {
	Name       string              `yaml:"name"`
	Protocol   string              `yaml:"protocol"`
	Credential declCredentialField `yaml:"credential,omitempty"`
	Enabled    *bool               `yaml:"enabled,omitempty"`
	Options    map[string]any      `yaml:"options,omitempty"`
}

// declCredentialField is either an inline secret map or a legacy name reference string.
type declCredentialField struct {
	Name string
	Data map[string]any
}

func (f *declCredentialField) UnmarshalYAML(value *yaml.Node) error {
	f.Name = ""
	f.Data = nil
	if value == nil || value.Kind == yaml.MappingNode {
		return value.Decode(&f.Data)
	}
	return value.Decode(&f.Name)
}

func (f declCredentialField) MarshalYAML() (any, error) {
	if len(f.Data) > 0 {
		return f.Data, nil
	}
	if f.Name != "" {
		return f.Name, nil
	}
	return nil, nil
}

type declTable struct {
	Server     string         `yaml:"server"`
	Schema     string         `yaml:"schema,omitempty"`
	Name       string         `yaml:"name"`
	RemoteName string         `yaml:"remoteName,omitempty"`
	KeyColumns []string       `yaml:"keyColumns,omitempty"`
	Options    map[string]any `yaml:"options,omitempty"`
	Columns    []declColumn   `yaml:"columns"`
}

type declColumn struct {
	Name       string `yaml:"name"`
	DataType   string `yaml:"dataType,omitempty"`
	RemoteName string `yaml:"remoteName,omitempty"`
	Nullable   *bool  `yaml:"nullable,omitempty"`
	Position   int    `yaml:"position,omitempty"`
}

type declFunction struct {
	Server     string         `yaml:"server"`
	Schema     string         `yaml:"schema,omitempty"`
	Name       string         `yaml:"name"`
	Operation  string         `yaml:"operation"`
	RemotePath string         `yaml:"remotePath,omitempty"`
	Method     string         `yaml:"method,omitempty"`
	Options    map[string]any `yaml:"options,omitempty"`
}

func isDeclarativeYAML(raw []byte) bool {
	var probe struct {
		Credentials []struct {
			Data    map[string]any `yaml:"data"`
			Payload string         `yaml:"payload"`
		} `yaml:"credentials"`
		Servers []struct {
			Credential yaml.Node `yaml:"credential"`
		} `yaml:"servers"`
		Tables []struct {
			Server   string    `yaml:"server"`
			ServerID uuid.UUID `yaml:"serverId"`
		} `yaml:"tables"`
	}
	if err := yaml.Unmarshal(raw, &probe); err != nil {
		return false
	}
	for _, c := range probe.Credentials {
		if len(c.Data) > 0 && c.Payload == "" {
			return true
		}
	}
	for _, s := range probe.Servers {
		if s.Credential.Kind == yaml.MappingNode {
			return true
		}
		if s.Credential.Kind == yaml.ScalarNode && s.Credential.Value != "" {
			return true
		}
	}
	for _, t := range probe.Tables {
		if t.Server != "" && t.ServerID == uuid.Nil {
			return true
		}
	}
	return false
}

func compileDeclarative(vault *auth.Vault, doc declarativeDoc) (snapshot, error) {
	now := time.Now().UTC()
	out := snapshot{}
	credIDs := map[string]uuid.UUID{}

	registerCred := func(name string, data map[string]any) (uuid.UUID, error) {
		name = strings.TrimSpace(name)
		if name == "" {
			return uuid.Nil, fmt.Errorf("credential name is required")
		}
		if id, ok := credIDs[name]; ok {
			return id, nil
		}
		payload, err := vault.Encrypt(expandEnvMap(data))
		if err != nil {
			return uuid.Nil, fmt.Errorf("credential %q: %w", name, err)
		}
		id := uuid.New()
		credIDs[name] = id
		out.Credentials = append(out.Credentials, credentialRecord{
			ID:        id,
			Name:      name,
			Payload:   encodePayload(payload),
			CreatedAt: now,
		})
		return id, nil
	}

	for _, c := range doc.Credentials {
		if strings.TrimSpace(c.Name) == "" {
			continue
		}
		if _, err := registerCred(c.Name, c.Data); err != nil {
			return snapshot{}, err
		}
	}

	serverIDs := map[string]uuid.UUID{}
	for _, s := range doc.Servers {
		name := strings.TrimSpace(s.Name)
		if name == "" || strings.TrimSpace(s.Protocol) == "" {
			return snapshot{}, fmt.Errorf("server requires name and protocol")
		}
		opts, err := mapToNode(s.Options)
		if err != nil {
			return snapshot{}, fmt.Errorf("server %q options: %w", name, err)
		}
		enabled := true
		if s.Enabled != nil {
			enabled = *s.Enabled
		}
		var credRef *uuid.UUID
		switch {
		case len(s.Credential.Data) > 0:
			credName := name
			id, err := registerCred(credName, s.Credential.Data)
			if err != nil {
				return snapshot{}, fmt.Errorf("server %q: %w", name, err)
			}
			credRef = &id
		case strings.TrimSpace(s.Credential.Name) != "":
			ref := strings.TrimSpace(s.Credential.Name)
			id, ok := credIDs[ref]
			if !ok {
				return snapshot{}, fmt.Errorf("server %q: unknown credential %q", name, ref)
			}
			credRef = &id
		}
		id := uuid.New()
		serverIDs[name] = id
		out.Servers = append(out.Servers, serverRecord{
			ID:            id,
			Name:          name,
			Protocol:      strings.TrimSpace(s.Protocol),
			Options:       opts,
			CredentialRef: credRef,
			Enabled:       enabled,
			UpdatedAt:     now,
		})
	}

	for _, t := range doc.Tables {
		serverName := strings.TrimSpace(t.Server)
		tableName := strings.TrimSpace(t.Name)
		if serverName == "" || tableName == "" {
			return snapshot{}, fmt.Errorf("table requires server and name")
		}
		srvID, ok := serverIDs[serverName]
		if !ok {
			return snapshot{}, fmt.Errorf("table %q: unknown server %q", tableName, serverName)
		}
		schema := strings.TrimSpace(t.Schema)
		if schema == "" {
			schema = "public"
		}
		opts, err := mapToNode(t.Options)
		if err != nil {
			return snapshot{}, fmt.Errorf("table %s.%s options: %w", schema, tableName, err)
		}
		keyCols := t.KeyColumns
		if keyCols == nil {
			keyCols = []string{}
		}
		tableID := uuid.New()
		out.Tables = append(out.Tables, tableRecord{
			ID:         tableID,
			ServerID:   srvID,
			SchemaName: schema,
			TableName:  tableName,
			RemoteName: t.RemoteName,
			KeyColumns: append([]string(nil), keyCols...),
			Options:    opts,
		})
		for i, c := range t.Columns {
			if strings.TrimSpace(c.Name) == "" {
				continue
			}
			dt := strings.TrimSpace(c.DataType)
			if dt == "" {
				dt = "text"
			}
			nullable := true
			if c.Nullable != nil {
				nullable = *c.Nullable
			}
			pos := c.Position
			if pos == 0 {
				pos = i
			}
			out.Columns = append(out.Columns, columnRecord{
				ID:         uuid.New(),
				TableID:    tableID,
				Name:       c.Name,
				DataType:   dt,
				RemoteName: c.RemoteName,
				Nullable:   nullable,
				Position:   pos,
			})
		}
	}

	for _, f := range doc.Functions {
		serverName := strings.TrimSpace(f.Server)
		name := strings.TrimSpace(f.Name)
		if serverName == "" || name == "" || strings.TrimSpace(f.Operation) == "" {
			return snapshot{}, fmt.Errorf("function requires server, name and operation")
		}
		srvID, ok := serverIDs[serverName]
		if !ok {
			return snapshot{}, fmt.Errorf("function %q: unknown server %q", name, serverName)
		}
		schema := strings.TrimSpace(f.Schema)
		if schema == "" {
			schema = "public"
		}
		opts, err := mapToNode(f.Options)
		if err != nil {
			return snapshot{}, fmt.Errorf("function %s.%s options: %w", schema, name, err)
		}
		out.Functions = append(out.Functions, functionRecord{
			ID:         uuid.New(),
			ServerID:   srvID,
			SchemaName: schema,
			Name:       name,
			Operation:  f.Operation,
			RemotePath: f.RemotePath,
			Method:     f.Method,
			Options:    opts,
		})
	}
	return out, nil
}

func decompileDeclarative(vault *auth.Vault, snap snapshot) (declarativeDoc, error) {
	doc := declarativeDoc{}
	credData := map[uuid.UUID]map[string]any{}
	serverNames := map[uuid.UUID]string{}

	for _, c := range snap.Credentials {
		payload, err := decodePayload(c.Payload)
		if err != nil {
			return declarativeDoc{}, err
		}
		data, err := vault.Decrypt(payload)
		if err != nil {
			return declarativeDoc{}, err
		}
		credData[c.ID] = data
	}

	refCount := map[uuid.UUID]int{}
	for _, s := range snap.Servers {
		if s.CredentialRef != nil {
			refCount[*s.CredentialRef]++
		}
	}

	for _, s := range snap.Servers {
		serverNames[s.ID] = s.Name
		var opts map[string]any
		_ = json.Unmarshal(nodeToRaw(s.Options), &opts)
		if opts == nil {
			opts = map[string]any{}
		}
		enabled := s.Enabled
		ds := declServer{
			Name:     s.Name,
			Protocol: s.Protocol,
			Enabled:  &enabled,
			Options:  opts,
		}
		if s.CredentialRef != nil {
			if data, ok := credData[*s.CredentialRef]; ok {
				if refCount[*s.CredentialRef] == 1 {
					ds.Credential = declCredentialField{Data: data}
				} else {
					ds.Credential = declCredentialField{Name: credentialNameForID(snap, *s.CredentialRef)}
				}
			}
		}
		doc.Servers = append(doc.Servers, ds)
	}

	for id, count := range refCount {
		if count > 1 {
			if data, ok := credData[id]; ok {
				doc.Credentials = append(doc.Credentials, declCredential{
					Name: credentialNameForID(snap, id),
					Data: data,
				})
			}
		}
	}

	tableMeta := map[string]uuid.UUID{}
	for _, t := range snap.Tables {
		serverName := serverNames[t.ServerID]
		tableMeta[t.SchemaName+"\x00"+t.TableName+"\x00"+serverName] = t.ID
		var opts map[string]any
		_ = json.Unmarshal(nodeToRaw(t.Options), &opts)
		if opts == nil {
			opts = map[string]any{}
		}
		doc.Tables = append(doc.Tables, declTable{
			Server:     serverName,
			Schema:     t.SchemaName,
			Name:       t.TableName,
			RemoteName: t.RemoteName,
			KeyColumns: append([]string(nil), t.KeyColumns...),
			Options:    opts,
		})
	}

	colsByTable := map[uuid.UUID][]declColumn{}
	for _, c := range snap.Columns {
		nullable := c.Nullable
		colsByTable[c.TableID] = append(colsByTable[c.TableID], declColumn{
			Name:       c.Name,
			DataType:   c.DataType,
			RemoteName: c.RemoteName,
			Nullable:   &nullable,
			Position:   c.Position,
		})
	}
	for i := range doc.Tables {
		key := doc.Tables[i].Schema + "\x00" + doc.Tables[i].Name + "\x00" + doc.Tables[i].Server
		if id, ok := tableMeta[key]; ok {
			doc.Tables[i].Columns = colsByTable[id]
		}
	}

	for _, f := range snap.Functions {
		var opts map[string]any
		_ = json.Unmarshal(nodeToRaw(f.Options), &opts)
		if opts == nil {
			opts = map[string]any{}
		}
		doc.Functions = append(doc.Functions, declFunction{
			Server:     serverNames[f.ServerID],
			Schema:     f.SchemaName,
			Name:       f.Name,
			Operation:  f.Operation,
			RemotePath: f.RemotePath,
			Method:     f.Method,
			Options:    opts,
		})
	}
	return doc, nil
}

func credentialNameForID(snap snapshot, id uuid.UUID) string {
	for _, c := range snap.Credentials {
		if c.ID == id {
			return c.Name
		}
	}
	return id.String()
}

func mapToNode(m map[string]any) (yaml.Node, error) {
	if len(m) == 0 {
		return rawToNode(json.RawMessage(`{}`)), nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return yaml.Node{}, err
	}
	return rawToNode(b), nil
}

func expandEnvMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = expandEnvValue(v)
	}
	return out
}

func expandEnvValue(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		key := strings.TrimSuffix(strings.TrimPrefix(s, "${"), "}")
		if ev := os.Getenv(key); ev != "" {
			return ev
		}
	}
	return v
}
