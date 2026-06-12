package cliapp

import (
	"fmt"
	"os"
)

// Run is the single CLI entry dispatcher (supabase-cli style).
func Run(args []string) int {
	if len(args) == 0 || isHelp(args[0]) {
		printUsage()
		return 2
	}

	switch args[0] {
	case "import":
		return runImport(args[1:])
	case "generate":
		return runGenerate(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printUsage()
		return 2
	}
}

func isHelp(arg string) bool {
	switch arg {
	case "help", "-h", "--help":
		return true
	default:
		return false
	}
}

func printUsage() {
	name := programName()
	fmt.Fprintf(os.Stderr, `%s — dataspan CLI

Usage:
  %s import   -input <file> [-output drivers.yaml] [-apply]
  %s generate types [typescript] [-url URL] [-schema public] [-output types.ts]

Commands:
  import    Convert OpenAPI / SQL / YAML into drivers.yaml (optional store apply)
  generate  Generate client artifacts from live API metadata

Environment:
  DATASPAN_URL       Server base URL (default http://localhost:3020)
  DATASPAN_ANON_KEY  apikey / Bearer for authenticated data API

Examples:
  %s import -input spec.yaml -output drivers.yaml
  %s import -input schema.sql -dialect postgres -apply
  %s generate types typescript -o database.types.ts
`, name, name, name, name, name, name)
}
