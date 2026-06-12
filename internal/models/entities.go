package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Meta DB entities (GORM model-driven schema).

type MetaCredential struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"uniqueIndex;not null"`
	Payload   []byte    `gorm:"not null"`
	CreatedAt time.Time
}

func (MetaCredential) TableName() string { return "credentials" }

type MetaServer struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Name          string         `gorm:"uniqueIndex;not null"`
	Protocol      Protocol       `gorm:"not null"`
	Options       datatypes.JSON `gorm:"not null;default:'{}'"`
	CredentialRef *uuid.UUID     `gorm:"type:uuid;constraint:OnDelete:SET NULL"`
	Enabled       bool           `gorm:"not null;default:true"`
	UpdatedAt     time.Time
}

func (MetaServer) TableName() string { return "servers" }

type MetaForeignTable struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey"`
	ServerID   uuid.UUID      `gorm:"type:uuid;not null;index"`
	SchemaName string         `gorm:"not null;default:public;uniqueIndex:idx_meta_tables_schema_table"`
	Name       string         `gorm:"column:table_name;not null;uniqueIndex:idx_meta_tables_schema_table"`
	RemoteName string
	KeyColumns datatypes.JSON `gorm:"not null;default:'[]'"`
	Options    datatypes.JSON `gorm:"not null;default:'{}'"`
	Server     MetaServer     `gorm:"foreignKey:ServerID;constraint:OnDelete:CASCADE"`
}

func (MetaForeignTable) TableName() string { return "tables" }

type MetaForeignColumn struct {
	ID         uuid.UUID        `gorm:"type:uuid;primaryKey"`
	TableID    uuid.UUID        `gorm:"type:uuid;not null;index"`
	Name       string           `gorm:"not null;uniqueIndex:idx_meta_columns_table_name"`
	DataType   string           `gorm:"not null;default:text"`
	RemoteName string
	Nullable   bool             `gorm:"not null;default:true"`
	Position   int              `gorm:"not null;default:0"`
	Table      MetaForeignTable `gorm:"foreignKey:TableID;constraint:OnDelete:CASCADE"`
}

func (MetaForeignColumn) TableName() string { return "columns" }

type MetaForeignFunction struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey"`
	ServerID   uuid.UUID      `gorm:"type:uuid;not null;index"`
	SchemaName string         `gorm:"not null;default:public;uniqueIndex:idx_meta_functions_schema_name"`
	Name       string         `gorm:"not null;uniqueIndex:idx_meta_functions_schema_name"`
	Operation  string         `gorm:"not null"`
	RemotePath string
	Method     string
	Options    datatypes.JSON `gorm:"not null;default:'{}'"`
	Server     MetaServer     `gorm:"foreignKey:ServerID;constraint:OnDelete:CASCADE"`
}

func (MetaForeignFunction) TableName() string { return "functions" }

// MetaModels returns all GORM models for AutoMigrate.
func MetaModels() []any {
	return []any{
		&MetaCredential{},
		&MetaServer{},
		&MetaForeignTable{},
		&MetaForeignColumn{},
		&MetaForeignFunction{},
	}
}
