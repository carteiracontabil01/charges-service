package supabase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/supabase-community/supabase-go"
)

var iamClient *supabase.Client
var companyClient *supabase.Client
var supabaseURL string
var supabaseKey string

func InitClient() {
	supabaseURL = strings.TrimSpace(os.Getenv("SUPABASE_URL"))
	supabaseKey = strings.TrimSpace(os.Getenv("SUPABASE_KEY"))

	iamClient = mustNewClient(supabaseURL, supabaseKey, "iam")
	companyClient = mustNewClient(supabaseURL, supabaseKey, "company")
}

func GetClient() *supabase.Client {
	// Backward-compat: default client is IAM
	return iamClient
}

func GetIAMClient() *supabase.Client {
	return iamClient
}

func GetCompanyClient() *supabase.Client {
	return companyClient
}

func mustNewClient(url, key, schema string) *supabase.Client {
	c, err := supabase.NewClient(url, key, &supabase.ClientOptions{
		Schema: schema,
		Headers: map[string]string{
			"X-Client-Info": "carteira-contabil-charges-service",
		},
	})
	if err != nil {
		log.Fatalf("Erro ao iniciar Supabase (schema=%s): %v", schema, err)
	}
	return c
}

// RpcPublic calls a PostgREST RPC under schema public (Accept-Profile/Content-Profile).
// This matches the pattern used in focus-integration-service and avoids depending on schema exposure.
func RpcPublic(name string, body any) (string, error) {
	if strings.TrimSpace(supabaseURL) == "" || strings.TrimSpace(supabaseKey) == "" {
		return "", fmt.Errorf("supabase env não configurado (SUPABASE_URL/SUPABASE_KEY)")
	}

	url := strings.TrimRight(supabaseURL, "/") + "/rest/v1/rpc/" + name

	var payload []byte
	if body == nil {
		payload = []byte(`{}`)
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("erro ao serializar payload RPC: %w", err)
		}
		payload = b
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("erro ao criar request RPC: %w", err)
	}

	req.Header.Set("Accept-Profile", "public")
	req.Header.Set("Content-Profile", "public")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apikey", supabaseKey)
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	req.Header.Set("X-Client-Info", "carteira-contabil-charges-service")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro ao executar RPC: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("rpc %s failed: HTTP %d: %s", name, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return strings.TrimSpace(string(respBody)), nil
}
