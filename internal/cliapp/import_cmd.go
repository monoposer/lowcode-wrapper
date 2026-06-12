package cliapp

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/importer"
	"github.com/monoposer/dataspan/internal/logx"
	"github.com/monoposer/dataspan/internal/store"
)

func runImport(args []string) int {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	kind := fs.String("kind", "", "input kind: openapi, sql, yaml (auto-detected when omitted)")
	input := fs.String("input", "", "input file path (- for stdin)")
	output := fs.String("output", "", "output yaml path (- for stdout)")
	dialect := fs.String("dialect", "", "sql dialect: postgres, mysql, sqlite")
	serverName := fs.String("server-name", "", "server name in generated yaml")
	endpoint := fs.String("endpoint", "", "http endpoint override for openapi")
	schema := fs.String("schema", "public", "default schema for sql import")
	mode := fs.String("mode", "replace", "store import mode: replace or merge")
	apply := fs.Bool("apply", false, "import into store (DATASPAN_STORE_MODE / .env)")
	_ = fs.Parse(args)
	if fs.NArg() > 0 {
		return exitUsage(fmt.Sprintf("import: unknown arguments: %s", strings.Join(fs.Args(), " ")))
	}
	if *input == "" {
		return exitUsage("import: -input <file> is required")
	}

	raw, err := readInput(*input)
	if err != nil {
		return exitError("import", "read input", err)
	}

	k := importer.Kind(strings.TrimSpace(*kind))
	if k == "" {
		k = importer.DetectKind(*input, raw)
	}

	opts := importer.ConvertOptions{Kind: k, Input: raw}
	switch k {
	case importer.KindOpenAPI:
		opts.OpenAPI = importer.OpenAPIOptions{
			ServerName: *serverName,
			Endpoint:   *endpoint,
		}
	case importer.KindSQL:
		d, err := resolveSQLDialect(*dialect)
		if err != nil {
			return exitError("import", "sql dialect", err)
		}
		opts.SQL = importer.SQLOptions{
			ServerName: *serverName,
			Schema:     *schema,
			Dialect:    d,
		}
	case importer.KindYAML:
	default:
		return exitError("import", "unsupported kind", fmt.Errorf("%q", k))
	}

	out, err := importer.Convert(opts)
	if err != nil {
		return exitError("import", "convert", err)
	}

	if *apply {
		if err := applyImportToStore(out, importer.ImportMode(*mode)); err != nil {
			return exitError("import", "apply to store", err)
		}
		logx.Component("import").Info("store import completed", "mode", *mode)
	}

	if err := writeOutput(*output, out); err != nil {
		return exitError("import", "write output", err)
	}
	if !*apply && *output != "" && *output != "-" {
		logx.Component("import").Info("wrote drivers yaml", "output", *output)
	}
	return 0
}

func resolveSQLDialect(raw string) (importer.SQLDialect, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if isTerminal(os.Stdin) {
			fmt.Fprint(os.Stderr, "SQL dialect (postgres/mysql/sqlite): ")
			var line string
			if _, err := fmt.Fscanln(os.Stdin, &line); err != nil {
				return "", fmt.Errorf("read dialect: %w", err)
			}
			raw = line
		} else {
			return "", fmt.Errorf("sql import requires -dialect postgres|mysql|sqlite")
		}
	}
	return importer.ParseSQLDialect(raw)
}

func applyImportToStore(yaml []byte, mode importer.ImportMode) error {
	vault, err := auth.NewVaultFromEnv()
	if err != nil {
		return err
	}
	st, err := store.NewFromEnv(vault)
	if err != nil {
		return err
	}
	defer st.Close()

	result, err := store.ImportYAML(context.Background(), st, yaml, mode)
	if err != nil {
		return err
	}
	logx.Component("import").Info("import stats",
		"servers", result.Servers,
		"tables", result.Tables,
		"functions", result.Functions,
	)
	return nil
}
