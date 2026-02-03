package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	CorsAllowedOrigins []string
}

func Load() Config {
	loadDotEnvBestEffort()

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8083"
	}

	origins := parseCSV(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if len(origins) == 0 {
		origins = []string{"http://localhost:4200"}
	}
	log.Printf("CORS_ALLOWED_ORIGINS=%q parsed=%v", os.Getenv("CORS_ALLOWED_ORIGINS"), origins)

	return Config{
		Port:               port,
		CorsAllowedOrigins: origins,
	}
}

func loadDotEnvBestEffort() {
	dir, err := os.Getwd()
	if err != nil {
		log.Println("Erro ao obter diretório atual:", err)
		return
	}

	envPath := filepath.Join(dir, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		envPath = filepath.Join(dir, "..", "..", ".env")
	}

	if err := godotenv.Load(envPath); err != nil {
		log.Println(".env not found, using system environment")
	}
}

func parseCSV(s string) []string {
	parts := strings.Split(strings.TrimSpace(s), ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
