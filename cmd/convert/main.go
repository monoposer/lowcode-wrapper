package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"lowcode-wrapper/internal/auth"
	"lowcode-wrapper/internal/importer"
	"lowcode-wrapper/internal/logx"
	"lowcode-wrapper/internal/store"
)

func main() {
	logx.Init()

	fs := flag.NewFlagSet("convert", flag.ExitOnError)
	kind := fs.String("kind", "", "input kind: openapi, sql, yaml (auto-detected from -input when omitted)")
	input := fs.String("input", "", "input file path (- for stdin)")
	output := fs.String("output", "", "output yaml path (- for stdout)")
	dialect := fs.String("dialect", "", "sql dialect: postgres, mysql, sqlite (required for sql)")
	serverName := fs.String("server-name", "", "server name in generated yaml")
	endpoint := fs.String("endpoint", "", "http endpoint override for openapi")
	schema := fs.String("schema", "public", "default schema for sql import")
	mode := fs.String("mode", "replace", "store import mode: replace or merge")
	apply := fs.Bool("apply", false, "import into store (WRAPPER_STORE_MODE / .env)")
	_ = fs.Parse(os.Args[1:])
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unknown arguments: %s\n", strings.Join(fs.Args(), " "))
		os.Exit(2)
	}

	if *input == "" {
		fmt.Fprintln(os.Stderr, "usage: convert -input <file> [-output drivers.yaml] [-kind openapi|sql] [-dialect pg] [-apply]")
		os.Exit(2)
	}

	raw, err := readInput(*input)
	if err != nil {
		fail(err)
	}

	k := importer.Kind(strings.TrimSpace(*kind))
	if k == "" {
		k = importer.DetectKind(*input, raw)
	}

	conv := importer.ConvertOptions{Kind: k, Input: raw}
	switch k {
	case importer.KindOpenAPI:
		conv.OpenAPI = importer.OpenAPIOptions{
			ServerName: *serverName,
			Endpoint:   *endpoint,
		}
	case importer.KindSQL:
		d, err := resolveDialect(*dialect)
		if err != nil {
			fail(err)
		}
		conv.SQL = importer.SQLOptions{
			ServerName: *serverName,
			Schema:     *schema,
			Dialect:    d,
		}
	case importer.KindYAML:
		// passthrough
	default:
		fail(fmt.Errorf("unsupported kind %q", k))
	}

	out, err := importer.Convert(conv)
	if err != nil {
		fail(err)
	}

	if *apply {
		if err := applyToStore(out, importer.ImportMode(*mode)); err != nil {
			fail(err)
		}
		logx.Component("convert").Info("import completed", "mode", *mode)
	}

	if err := writeOutput(*output, out); err != nil {
		fail(err)
	}
	if !*apply && *output != "" && *output != "-" {
		logx.Component("convert").Info("wrote drivers yaml", "output", *output)
	}
}

func readInput(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func writeOutput(path string, data []byte) error {
	if path == "" || path == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func resolveDialect(raw string) (importer.SQLDialect, error) {
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

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func applyToStore(yaml []byte, mode importer.ImportMode) error {
	vault, err := auth.NewVaultFromEnv()
	if err != nil {
		return err
	}
	st, err := store.NewFromEnv(vault)
	if err != nil {
		return err
	}
	defer st.Close()

	ctx := context.Background()
	result, err := store.ImportYAML(ctx, st, yaml, mode)
	if err != nil {
		return err
	}
	logx.Component("convert").Info("import stats",
		"servers", result.Servers,
		"tables", result.Tables,
		"functions", result.Functions,
	)
	return nil
}

func fail(err error) {
	slog.Error("convert failed", "err", err)
	os.Exit(1)
}
