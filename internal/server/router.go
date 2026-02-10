package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/seuuser/charges-service/internal/config"
	"github.com/seuuser/charges-service/internal/handler"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func RegisterRoutes(r *chi.Mux, cfg config.Config) {
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CorsAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Origin", "X-Requested-With", "X-Client-Info", "apikey", "asaas-access-token"},
		ExposedHeaders:   []string{"X-Total-Count", "Rate-Limit-Limit", "Rate-Limit-Remaining", "Rate-Limit-Reset"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health only (for now)
	r.Get("/health", handler.Health)

	// Swagger UI
	// Register both variants to avoid redirects (which can break some browser/CORS setups)
	r.Get("/swagger", httpSwagger.WrapHandler)
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Asaas Webhook (fee charges)
	r.Post("/asaas/feecharges", handler.ReceiveAsaasWebhook)

	// Asaas (initial)
	r.Post("/v1/asaas/customers", handler.CreateAsaasCustomer)
	r.Post("/v1/asaas/charges", handler.CreateAsaasCharge)
	r.Get("/v1/asaas/charges", handler.ListAsaasCharges)
	r.Put("/v1/asaas/charges/{id}", handler.UpdateAsaasCharge)
	r.Delete("/v1/asaas/charges/{id}", handler.DeleteAsaasCharge)
	r.Get("/v1/asaas/charges/{id}/digitable-line", handler.GetAsaasChargeDigitableLine)
	r.Get("/v1/asaas/charges/{id}/pix-qrcode", handler.GetAsaasChargePixQrCode)
	r.Get("/v1/asaas/customers/by-company", handler.GetAsaasCustomerByCompanyID)
	r.Get("/v1/asaas/customers/{id}", handler.GetAsaasCustomerByID)
	r.Put("/v1/asaas/customers/by-company", handler.UpdateAsaasCustomerByCompanyID)
	r.Put("/v1/asaas/customers/{id}", handler.UpdateAsaasCustomerByID)
}
