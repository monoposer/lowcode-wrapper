package main

import (
	"log"
	"net/http"
	"os"

	"lowcode-wrapper/internal/api"
	"lowcode-wrapper/internal/auth"
	"lowcode-wrapper/internal/service"
	store "lowcode-wrapper/internal/store/postgres"

	_ "lowcode-wrapper/internal/driver/filedriver"
	_ "lowcode-wrapper/internal/driver/httpdriver"
	_ "lowcode-wrapper/internal/driver/mysqldriver"
	_ "lowcode-wrapper/internal/driver/pgdriver"
)

func main() {
	vault, err := auth.NewVaultFromEnv()
	if err != nil {
		log.Fatalf("vault: %v", err)
	}
	s, err := store.NewFromEnv(vault)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer s.Close()

	engine := service.NewEngine(s)
	mux := http.NewServeMux()
	api.NewAdminHandler(s).Register(mux)
	api.NewPostgRESTHandler(engine).Register(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3020"
	}
	log.Printf("lowcode-wrapper listening on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, api.CORS(mux)); err != nil {
		log.Fatal(err)
	}
}
