package driver

import (
	"context"

	"lowcode-wrapper/internal/models"
	"lowcode-wrapper/internal/postgrest"
)

type SelectRequest struct {
	Resolved *models.ResolvedTable
	Select   []string
	Filters  []postgrest.Filter
	Order    []postgrest.OrderSpec
	Limit    int
	Offset   int
}

type RowRequest struct {
	Resolved             *models.ResolvedTable
	Row                  map[string]any
	Filters              []postgrest.Filter
	PreferRepresentation bool
	PreferUpsert         bool
	Returning            []string
}

type DeleteRequest struct {
	Resolved *models.ResolvedTable
	Filters  []postgrest.Filter
}

type InvokeRequest struct {
	Resolved *models.ResolvedFunction
	Body     map[string]any
	Query    map[string]string
}

type Driver interface {
	Select(ctx context.Context, req SelectRequest) ([]map[string]any, error)
	Insert(ctx context.Context, req RowRequest) (map[string]any, error)
	Update(ctx context.Context, req RowRequest) (int, error)
	Upsert(ctx context.Context, req RowRequest) (created bool, returned map[string]any, err error)
	Delete(ctx context.Context, req DeleteRequest) (int, error)
	Invoke(ctx context.Context, req InvokeRequest) (any, error)
}

type Factory func(ctx context.Context, srv models.Server, cred map[string]any) (Driver, error)
