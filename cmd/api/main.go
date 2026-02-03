package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/seuuser/charges-service/docs"
	"github.com/seuuser/charges-service/internal/config"
	"github.com/seuuser/charges-service/internal/server"
	"github.com/seuuser/charges-service/internal/supabase"
)

// @title           Charges Service
// @version         0.1.0
// @description     Microservice para gestão de cobranças (inicialmente integração Asaas; no futuro bancos diversos).
// @host            localhost:8083
// @BasePath        /
func main() {
	cfg := config.Load()
	supabase.InitClient()

	// Swagger host override (same pattern as other services)
	swaggerHost := strings.TrimSpace(os.Getenv("SWAGGER_HOST"))
	if swaggerHost == "" {
		swaggerHost = strings.TrimSpace(os.Getenv("PUBLIC_HOST"))
	}
	if swaggerHost == "" {
		swaggerHost = "localhost:" + cfg.Port
	}
	docs.SwaggerInfo.Host = swaggerHost
	docs.SwaggerInfo.BasePath = "/"

	if schemes := strings.TrimSpace(os.Getenv("SWAGGER_SCHEMES")); schemes != "" {
		parts := strings.Split(schemes, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			docs.SwaggerInfo.Schemes = out
		}
	}

	r := chi.NewRouter()
	// Avoid panics turning into "CORS errors" in browsers.
	r.Use(middleware.Recoverer)

	server.RegisterRoutes(r, cfg)

	log.Printf("listening on :%s …", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}

