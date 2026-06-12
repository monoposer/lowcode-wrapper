package db

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/monoposer/dataspan/internal/models"
)

func encodeKeyColumns(cols []string) datatypes.JSON {
	if cols == nil {
		return datatypes.JSON("[]")
	}
	b, _ := json.Marshal(cols)
	return datatypes.JSON(b)
}

func decodeKeyColumns(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var cols []string
	if err := json.Unmarshal(raw, &cols); err != nil || cols == nil {
		return []string{}
	}
	return cols
}

func rawJSON(raw datatypes.JSON) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(raw)
}

func toJSONRaw(raw json.RawMessage) datatypes.JSON {
	if len(raw) == 0 {
		return datatypes.JSON("{}")
	}
	return datatypes.JSON(raw)
}

func toCredential(c models.MetaCredential) models.Credential {
	return models.Credential{
		ID:        c.ID,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
	}
}

func toServer(s models.MetaServer) models.Server {
	return models.Server{
		ID:            s.ID,
		Name:          s.Name,
		Protocol:      s.Protocol,
		Options:       rawJSON(s.Options),
		CredentialRef: s.CredentialRef,
		Enabled:       s.Enabled,
		UpdatedAt:     s.UpdatedAt,
	}
}

func toTable(t models.MetaForeignTable, serverName string) models.Table {
	return models.Table{
		ID:         t.ID,
		ServerID:   t.ServerID,
		SchemaName: t.SchemaName,
		TableName:  t.Name,
		RemoteName: t.RemoteName,
		KeyColumns: decodeKeyColumns(t.KeyColumns),
		Options:    rawJSON(t.Options),
		ServerName: serverName,
	}
}

func toColumn(c models.MetaForeignColumn) models.Column {
	return models.Column{
		ID:         c.ID,
		TableID:    c.TableID,
		Name:       c.Name,
		DataType:   c.DataType,
		RemoteName: c.RemoteName,
		Nullable:   c.Nullable,
		Position:   c.Position,
	}
}

func toFunction(f models.MetaForeignFunction, serverName string) models.Function {
	return models.Function{
		ID:         f.ID,
		ServerID:   f.ServerID,
		SchemaName: f.SchemaName,
		Name:       f.Name,
		Operation:  f.Operation,
		RemotePath: f.RemotePath,
		Method:     f.Method,
		Options:    rawJSON(f.Options),
		ServerName: serverName,
	}
}

func newUUID() uuid.UUID {
	return uuid.New()
}
