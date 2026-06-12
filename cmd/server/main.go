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

	_ "lowcode-wrapper/internal/driver/filedriver"
	_ "lowcode-wrapper/internal/driver/firebasedriver"
	_ "lowcode-wrapper/internal/driver/httpdriver"
	_ "lowcode-wrapper/internal/driver/mongodriver"
	_ "lowcode-wrapper/internal/driver/mysqldriver"
	_ "lowcode-wrapper/internal/driver/notiondriver"
	_ "lowcode-wrapper/internal/driver/pgdriver"
	_ "lowcode-wrapper/internal/driver/redisdriver"
	_ "lowcode-wrapper/internal/driver/s3driver"
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
