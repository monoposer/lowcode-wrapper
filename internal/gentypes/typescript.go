package gentypes

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateTypeScript emits Supabase-compatible Database types from a schema snapshot.
func GenerateTypeScript(snap *SchemaSnapshot) string {
	if snap == nil || len(snap.Schemas) == 0 {
		return strings.TrimSpace(`export type Json =
  | string
  | number
  | boolean
  | null
  | { [key: string]: Json | undefined }
  | Json[]

export interface Database {
  public: {
    Tables: Record<string, never>
    Functions: Record<string, never>
  }
}
`) + "\n"
	}

	var b strings.Builder
	b.WriteString(`export type Json =
  | string
  | number
  | boolean
  | null
  | { [key: string]: Json | undefined }
  | Json[]

export interface Database {
`)
	schemaNames := sortedKeys(snap.Schemas)
	for _, schema := range schemaNames {
		st := snap.Schemas[schema]
		fmt.Fprintf(&b, "  %s: {\n", tsIdent(schema))
		b.WriteString("    Tables: {\n")
		tableNames := sortedTableKeys(st.Tables)
		for _, tableName := range tableNames {
			writeTableTypes(&b, st.Tables[tableName])
		}
		b.WriteString("    }\n")
		b.WriteString("    Functions: {\n")
		fnNames := sortedFunctionKeys(st.Functions)
		for _, fnName := range fnNames {
			writeFunctionTypes(&b, st.Functions[fnName])
		}
		b.WriteString("    }\n")
		b.WriteString("  }\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func writeTableTypes(b *strings.Builder, tm TableMeta) {
	cols := append([]ColumnMeta(nil), tm.Columns...)
	sort.Slice(cols, func(i, j int) bool {
		if cols[i].Position != cols[j].Position {
			return cols[i].Position < cols[j].Position
		}
		return cols[i].Name < cols[j].Name
	})
	keySet := map[string]bool{}
	for _, k := range tm.KeyColumns {
		keySet[k] = true
	}

	fmt.Fprintf(b, "      %s: {\n", tsIdent(tm.TableName))
	b.WriteString("        Row: {\n")
	for _, col := range cols {
		writeField(b, "          ", col.Name, tsRowType(col), false)
	}
	b.WriteString("        }\n")
	b.WriteString("        Insert: {\n")
	for _, col := range cols {
		optional := col.Nullable || !keySet[col.Name]
		writeField(b, "          ", col.Name, tsInsertType(col), optional)
	}
	b.WriteString("        }\n")
	b.WriteString("        Update: {\n")
	for _, col := range cols {
		writeField(b, "          ", col.Name, tsRowType(col), true)
	}
	b.WriteString("        }\n")
	b.WriteString("      }\n")
}

func writeFunctionTypes(b *strings.Builder, fn FunctionMeta) {
	fmt.Fprintf(b, "      %s: {\n", tsIdent(fn.Name))
	b.WriteString("        Args: Record<string, Json | undefined>\n")
	b.WriteString("        Returns: Json\n")
	b.WriteString("      }\n")
}

func writeField(b *strings.Builder, indent, name, typ string, optional bool) {
	if optional {
		fmt.Fprintf(b, "%s%s?: %s\n", indent, tsIdent(name), typ)
	} else {
		fmt.Fprintf(b, "%s%s: %s\n", indent, tsIdent(name), typ)
	}
}

func tsRowType(col ColumnMeta) string {
	base := tsDataType(col.DataType)
	if col.Nullable {
		return base + " | null"
	}
	return base
}

func tsInsertType(col ColumnMeta) string {
	return tsRowType(col)
}

func tsDataType(dataType string) string {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "integer", "int", "bigint", "smallint", "numeric", "number", "float", "double", "decimal":
		return "number"
	case "boolean", "bool":
		return "boolean"
	case "json", "jsonb", "object":
		return "Json"
	case "array":
		return "Json[]"
	default:
		return "string"
	}
}

func tsIdent(name string) string {
	if name == "" {
		return "_"
	}
	if isValidTSIdent(name) {
		return name
	}
	return fmt.Sprintf("%q", name)
}

func isValidTSIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				return false
			}
			continue
		}
		if r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]SchemaTables) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedTableKeys(m map[string]TableMeta) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedFunctionKeys(m map[string]FunctionMeta) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
