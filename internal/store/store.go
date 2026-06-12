package store

import (
	"context"

	"github.com/google/uuid"

	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/models"
)

type Store interface {
	Close()
	Vault() *auth.Vault

	CreateCredential(ctx context.Context, name string, data map[string]any) (*models.Credential, error)
	DeleteCredential(ctx context.Context, id uuid.UUID) error
	ResolveCredential(ctx context.Context, ref uuid.UUID) (map[string]any, error)

	CreateServer(ctx context.Context, req models.CreateServerRequest) (*models.Server, error)
	ListServers(ctx context.Context) ([]models.Server, error)
	GetServerByID(ctx context.Context, id uuid.UUID) (*models.Server, error)
	GetServerByName(ctx context.Context, name string) (*models.Server, error)
	UpdateServer(ctx context.Context, id uuid.UUID, req models.UpdateServerRequest) (*models.Server, error)
	DeleteServer(ctx context.Context, id uuid.UUID) error

	CreateTable(ctx context.Context, req models.CreateTableRequest) (*models.Table, []models.Column, error)
	ListTables(ctx context.Context) ([]models.Table, error)
	GetTableByID(ctx context.Context, id uuid.UUID) (*models.Table, error)
	UpdateTable(ctx context.Context, id uuid.UUID, req models.UpdateTableRequest) (*models.Table, error)
	DeleteTable(ctx context.Context, id uuid.UUID) error
	ListColumns(ctx context.Context, schema, table string) ([]models.Column, error)
	ListColumnsByTableID(ctx context.Context, tableID uuid.UUID) ([]models.Column, error)
	GetColumnByID(ctx context.Context, id uuid.UUID) (*models.Column, error)
	CreateColumn(ctx context.Context, tableID uuid.UUID, req models.CreateColumnRequest) (*models.Column, error)
	UpdateColumn(ctx context.Context, id uuid.UUID, req models.UpdateColumnRequest) (*models.Column, error)
	DeleteColumn(ctx context.Context, id uuid.UUID) error
	ResolveTable(ctx context.Context, schema, table string) (*models.ResolvedTable, error)

	CreateFunction(ctx context.Context, req models.CreateFunctionRequest) (*models.Function, error)
	ListFunctions(ctx context.Context) ([]models.Function, error)
	GetFunctionByID(ctx context.Context, id uuid.UUID) (*models.Function, error)
	UpdateFunction(ctx context.Context, id uuid.UUID, req models.UpdateFunctionRequest) (*models.Function, error)
	DeleteFunction(ctx context.Context, id uuid.UUID) error
	ResolveFunction(ctx context.Context, schema, name string) (*models.ResolvedFunction, error)

	ServerCredential(ctx context.Context, srv *models.Server) (map[string]any, error)
}
