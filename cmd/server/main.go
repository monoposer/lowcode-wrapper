package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/monoposer/dataspan/internal/api"
	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/logx"
	"github.com/monoposer/dataspan/internal/service"
	"github.com/monoposer/dataspan/internal/store"
	"github.com/monoposer/dataspan/internal/version"

	_ "github.com/monoposer/dataspan/internal/driver/airtable"
	_ "github.com/monoposer/dataspan/internal/driver/file"
	_ "github.com/monoposer/dataspan/internal/driver/firebase"
	_ "github.com/monoposer/dataspan/internal/driver/http"
	_ "github.com/monoposer/dataspan/internal/driver/mongo"
	_ "github.com/monoposer/dataspan/internal/driver/mysql"
	_ "github.com/monoposer/dataspan/internal/driver/notion"
	_ "github.com/monoposer/dataspan/internal/driver/postgres"
	_ "github.com/monoposer/dataspan/internal/driver/redis"
	_ "github.com/monoposer/dataspan/internal/driver/s3"
	_ "github.com/monoposer/dataspan/internal/driver/sheets"
)

func main() {
	logx.Init()

	vault, err := auth.NewVaultFromEnv()
	if err != nil {
		slog.Error("vault init failed", "err", err)
		os.Exit(1)
	}
	s, err := store.NewFromEnv(vault)
	if err != nil {
		slog.Error("meta store init failed", "err", err)
		os.Exit(1)
	}
	defer s.Close()
	logx.Component("server").Info("meta store ready")

	engine := service.NewEngine(s)
	gateway := auth.NewGatewayFromEnv()
	if gateway.Enabled {
		logx.Component("server").Info("data API auth enabled")
	}

	mux := http.NewServeMux()
	api.RegisterOpenAPI(mux)
	api.NewAdminHandler(s).Register(mux)
	api.NewPostgRESTHandler(engine).Register(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3020"
	}
	handler := api.CORS(api.Logging(api.DataAPIAuth(gateway, mux)))
	logx.Component("server").Info("listening",
		"version", version.Version,
		"addr", "http://localhost:"+port,
		"swagger", "/swagger/",
	)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
