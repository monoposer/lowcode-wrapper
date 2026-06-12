package main

import (
	"log/slog"
	"net/http"
	"os"

	"lowcode-wrapper/internal/api"
	"lowcode-wrapper/internal/auth"
	"lowcode-wrapper/internal/logx"
	"lowcode-wrapper/internal/service"
	"lowcode-wrapper/internal/store"

	_ "lowcode-wrapper/internal/driver/airtable"
	_ "lowcode-wrapper/internal/driver/file"
	_ "lowcode-wrapper/internal/driver/firebase"
	_ "lowcode-wrapper/internal/driver/http"
	_ "lowcode-wrapper/internal/driver/mongo"
	_ "lowcode-wrapper/internal/driver/mysql"
	_ "lowcode-wrapper/internal/driver/notion"
	_ "lowcode-wrapper/internal/driver/postgres"
	_ "lowcode-wrapper/internal/driver/redis"
	_ "lowcode-wrapper/internal/driver/s3"
	_ "lowcode-wrapper/internal/driver/sheets"
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
	mux := http.NewServeMux()
	api.RegisterPlayground(mux)
	api.RegisterOpenAPI(mux)
	api.NewAdminHandler(s).Register(mux)
	api.NewPostgRESTHandler(engine).Register(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3020"
	}
	handler := api.CORS(api.Logging(mux))
	logx.Component("server").Info("listening",
		"addr", "http://localhost:"+port,
		"playground", "/playground/",
		"swagger", "/swagger/",
	)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
