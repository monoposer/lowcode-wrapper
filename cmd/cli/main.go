package main

import (
	"os"

	"github.com/monoposer/dataspan/internal/cliapp"
	"github.com/monoposer/dataspan/internal/logx"
)

func main() {
	logx.Init()
	os.Exit(cliapp.Run(os.Args[1:]))
}
