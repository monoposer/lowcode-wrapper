package cliapp

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/monoposer/dataspan/internal/gentypes"
	"github.com/monoposer/dataspan/internal/logx"
)

func runGenerate(args []string) int {
	if len(args) == 0 {
		return exitUsage("generate: subcommand required (supported: types)")
	}
	switch args[0] {
	case "types":
		return runGenerateTypes(args[1:])
	default:
		return exitUsage(fmt.Sprintf("generate: unknown subcommand %q (supported: types)", args[0]))
	}
}

func runGenerateTypes(args []string) int {
	lang := parseTypeScriptLangArg(&args)

	fs := flag.NewFlagSet("generate types", flag.ExitOnError)
	langFlag := fs.String("lang", lang, "output language: typescript")
	url := fs.String("url", envOr("DATASPAN_URL", "http://localhost:3020"), "dataspan server base URL")
	schema := fs.String("schema", "", "comma-separated schema profiles (default: all)")
	output := fs.String("output", "-", "output file (- for stdout)")
	apikey := fs.String("apikey", os.Getenv("DATASPAN_ANON_KEY"), "apikey header")
	token := fs.String("token", "", "Bearer token (defaults to apikey)")
	_ = fs.Parse(args)
	if fs.NArg() > 0 {
		return exitUsage(fmt.Sprintf("generate types: unknown arguments: %s", strings.Join(fs.Args(), " ")))
	}

	lang = normalizeTypeScriptLang(*langFlag)
	if lang == "" {
		return exitUsage(fmt.Sprintf("generate types: unsupported lang %q (supported: typescript)", *langFlag))
	}

	bearer := strings.TrimSpace(*token)
	if bearer == "" {
		bearer = strings.TrimSpace(*apikey)
	}

	client := &gentypes.MetaClient{
		BaseURL: *url,
		APIKey:  strings.TrimSpace(*apikey),
		Token:   bearer,
	}

	snap, err := client.Fetch(context.Background(), splitCSV(*schema))
	if err != nil {
		return exitError("generate", "fetch metadata", err)
	}

	out := gentypes.GenerateTypeScript(snap)
	if err := writeOutput(*output, []byte(out)); err != nil {
		return exitError("generate", "write output", err)
	}
	if *output != "" && *output != "-" {
		logx.Component("generate").Info("wrote types", "output", *output, "schemas", len(snap.Schemas))
	}
	return 0
}

func parseTypeScriptLangArg(args *[]string) string {
	if len(*args) == 0 {
		return "typescript"
	}
	switch strings.ToLower(strings.TrimSpace((*args)[0])) {
	case "typescript", "ts":
		*args = (*args)[1:]
		return "typescript"
	default:
		return "typescript"
	}
}

func normalizeTypeScriptLang(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "typescript", "ts":
		return "typescript"
	default:
		return ""
	}
}
