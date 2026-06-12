package gentypes

import (
	"strings"
	"testing"
)

func TestGenerateTypeScript(t *testing.T) {
	snap := &SchemaSnapshot{
		Schemas: map[string]SchemaTables{
			"public": {
				Tables: map[string]TableMeta{
					"orders": {
						SchemaName: "public",
						TableName:  "orders",
						KeyColumns: []string{"id"},
						Columns: []ColumnMeta{
							{Name: "id", DataType: "text", Nullable: false, Position: 1},
							{Name: "amount", DataType: "numeric", Nullable: true, Position: 2},
						},
					},
				},
				Functions: map[string]FunctionMeta{
					"ship": {Name: "ship", Operation: "invoke"},
				},
			},
		},
	}

	out := GenerateTypeScript(snap)
	for _, want := range []string{
		"export interface Database",
		"orders: {",
		"id: string",
		"amount: number | null",
		"id: string",
		"amount?: number | null",
		"ship: {",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
