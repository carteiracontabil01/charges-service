# Charges Service (Gestão de Cobranças)

Microserviço Go para gestão de cobranças do escritório contábil.

Fase 1 (agora):
- Health check: `GET /health`

Fases futuras:
- Integração Asaas (criar/editar/cancelar cobranças)
- Webhook de pagamento (eventos, conciliação)
- Outras integrações (bancos, etc), por pasta em `internal/integrations/`

## Rodar localmente

```bash
cp env.example .env
make run
```

## Health

```bash
curl -s http://localhost:8083/health
```

