package engine

import (
	"context"

	"github.com/google/uuid"

	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/models"
)

type metaStoreStub struct {
	servers   []models.Server
	tables    []models.Table
	functions []models.Function
	listCalls int
}

func (m *metaStoreStub) Close()                                                  {}
func (m *metaStoreStub) Vault() *auth.Vault                                      { return nil }
func (m *metaStoreStub) CreateCredential(context.Context, string, map[string]any) (*models.Credential, error) {
	panic("unused")
}
func (m *metaStoreStub) DeleteCredential(context.Context, uuid.UUID) error { panic("unused") }
func (m *metaStoreStub) ResolveCredential(context.Context, uuid.UUID) (map[string]any, error) {
	panic("unused")
}
func (m *metaStoreStub) CreateServer(context.Context, models.CreateServerRequest) (*models.Server, error) {
	panic("unused")
}
func (m *metaStoreStub) ListServers(context.Context) ([]models.Server, error) {
	m.listCalls++
	return m.servers, nil
}
func (m *metaStoreStub) GetServerByID(context.Context, uuid.UUID) (*models.Server, error) {
	panic("unused")
}
func (m *metaStoreStub) GetServerByName(context.Context, string) (*models.Server, error) {
	panic("unused")
}
func (m *metaStoreStub) UpdateServer(context.Context, uuid.UUID, models.UpdateServerRequest) (*models.Server, error) {
	panic("unused")
}
func (m *metaStoreStub) DeleteServer(context.Context, uuid.UUID) error { panic("unused") }
func (m *metaStoreStub) CreateTable(context.Context, models.CreateTableRequest) (*models.Table, []models.Column, error) {
	panic("unused")
}
func (m *metaStoreStub) ListTables(context.Context) ([]models.Table, error) {
	m.listCalls++
	return m.tables, nil
}
func (m *metaStoreStub) GetTableByID(context.Context, uuid.UUID) (*models.Table, error) {
	panic("unused")
}
func (m *metaStoreStub) UpdateTable(context.Context, uuid.UUID, models.UpdateTableRequest) (*models.Table, error) {
	panic("unused")
}
func (m *metaStoreStub) DeleteTable(context.Context, uuid.UUID) error { panic("unused") }
func (m *metaStoreStub) ListColumns(context.Context, string, string) ([]models.Column, error) {
	panic("unused")
}
func (m *metaStoreStub) ListColumnsByTableID(context.Context, uuid.UUID) ([]models.Column, error) {
	panic("unused")
}
func (m *metaStoreStub) GetColumnByID(context.Context, uuid.UUID) (*models.Column, error) {
	panic("unused")
}
func (m *metaStoreStub) CreateColumn(context.Context, uuid.UUID, models.CreateColumnRequest) (*models.Column, error) {
	panic("unused")
}
func (m *metaStoreStub) UpdateColumn(context.Context, uuid.UUID, models.UpdateColumnRequest) (*models.Column, error) {
	panic("unused")
}
func (m *metaStoreStub) DeleteColumn(context.Context, uuid.UUID) error { panic("unused") }
func (m *metaStoreStub) ResolveTable(context.Context, string, string) (*models.ResolvedTable, error) {
	panic("unused")
}
func (m *metaStoreStub) CreateFunction(context.Context, models.CreateFunctionRequest) (*models.Function, error) {
	panic("unused")
}
func (m *metaStoreStub) ListFunctions(context.Context) ([]models.Function, error) {
	m.listCalls++
	return m.functions, nil
}
func (m *metaStoreStub) GetFunctionByID(context.Context, uuid.UUID) (*models.Function, error) {
	panic("unused")
}
func (m *metaStoreStub) UpdateFunction(context.Context, uuid.UUID, models.UpdateFunctionRequest) (*models.Function, error) {
	panic("unused")
}
func (m *metaStoreStub) DeleteFunction(context.Context, uuid.UUID) error { panic("unused") }
func (m *metaStoreStub) ResolveFunction(context.Context, string, string) (*models.ResolvedFunction, error) {
	panic("unused")
}
func (m *metaStoreStub) ServerCredential(context.Context, *models.Server) (map[string]any, error) {
	panic("unused")
}
