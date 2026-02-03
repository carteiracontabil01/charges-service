# Configuração de Webhook Asaas no charges-service

Este documento explica como configurar webhooks do Asaas para receber atualizações automáticas de status de cobranças.

## 📋 Visão Geral

O `charges-service` possui um endpoint `/asaas/feecharges` que recebe eventos do Asaas sempre que ocorre uma mudança no status de uma cobrança (pagamento recebido, vencida, cancelada, etc.).

## 🔐 Segurança

O webhook utiliza validação via header `asaas-access-token` para garantir que apenas o Asaas possa enviar eventos.

### Gerando um Token Seguro

1. Gere um UUID v4 forte em: https://www.uuidgenerator.net/version4
2. Configure este token no arquivo `.env`:

```bash
ASAAS_WEBHOOK_SECRET=seu-uuid-v4-aqui
```

3. Use o **mesmo token** ao configurar o webhook no painel do Asaas

## 🌐 Configurando o Webhook no Asaas

### Via Aplicação Web (Recomendado para Teste)

1. Acesse o painel do Asaas (Sandbox ou Produção)
2. Vá em **Menu do Usuário** → **Integrações** → **Webhooks**
3. Clique em **Novo Webhook**
4. Configure:
   - **URL**: `https://seu-dominio.com/asaas/feecharges`
     - **Local/HML**: `https://xxxx.ngrok-free.app/asaas/feecharges`
     - **PRD**: `https://api-charges.carteiracontabil.com/asaas/feecharges`
   - **Eventos**: Selecione todos os eventos de **Cobranças** (PAYMENT_*)
   - **Header de Autenticação**: `asaas-access-token`
   - **Valor do Header**: Cole o UUID v4 gerado anteriormente
   - **Tipo de Envio**: `Sequencial` (padrão)

5. Salve o webhook

### Via API

```bash
curl --location 'https://sandbox.asaas.com/api/v3/webhooks' \
--header 'access_token: SEU_TOKEN_ASAAS' \
--header 'Content-Type: application/json' \
--data '{
  "name": "Charges Service Webhook",
  "url": "https://xxxx.ngrok-free.app/asaas/feecharges",
  "email": "seu-email@empresa.com",
  "sendType": "SEQUENTIALLY",
  "apiVersion": 3,
  "enabled": true,
  "interrupted": false,
  "authToken": "seu-uuid-v4-aqui",
  "events": [
    "PAYMENT_CREATED",
    "PAYMENT_UPDATED",
    "PAYMENT_CONFIRMED",
    "PAYMENT_RECEIVED",
    "PAYMENT_OVERDUE",
    "PAYMENT_DELETED",
    "PAYMENT_RESTORED",
    "PAYMENT_REFUNDED",
    "PAYMENT_RECEIVED_IN_CASH",
    "PAYMENT_CHARGEBACK_REQUESTED",
    "PAYMENT_CHARGEBACK_DISPUTE",
    "PAYMENT_AWAITING_CHARGEBACK_REVERSAL",
    "PAYMENT_DUNNING_RECEIVED",
    "PAYMENT_DUNNING_REQUESTED",
    "PAYMENT_BANK_SLIP_VIEWED",
    "PAYMENT_CHECKOUT_VIEWED"
  ]
}
'
```

## 🧪 Testando o Webhook (Sandbox)

### 1. Configurando o ambiente local

Para testar localmente, você precisa expor seu localhost para a internet. Use **ngrok** ou **Cloudflare Tunnel**:

#### Com ngrok

```bash
ngrok http 8083
```

Copie a URL gerada (ex: `https://abc123.ngrok-free.app`) e configure no webhook:
- URL: `https://abc123.ngrok-free.app/asaas/feecharges`

#### Com Cloudflare Tunnel

```bash
cloudflared tunnel --url http://localhost:8083
```

### 2. Criando uma cobrança de teste

```bash
curl --location 'http://localhost:8083/v1/asaas/charges' \
--header 'Content-Type: application/json' \
--data '{
  "companyId": "uuid-da-empresa",
  "contractId": "uuid-do-contrato",
  "payload": {
    "billingType": "BOLETO",
    "value": 100.00,
    "dueDate": "2026-02-15",
    "description": "Teste Webhook"
  }
}
'
```

### 3. Simulando pagamento (Sandbox)

No painel do Asaas Sandbox:
1. Vá em **Cobranças**
2. Localize a cobrança criada
3. Clique em **Ações** → **Confirmar Pagamento**

Isso dispara o evento `PAYMENT_RECEIVED` para o webhook.

### 4. Verificando os logs

No terminal do `charges-service`, você verá:

```
[webhook] ✅ Received event: PAYMENT_RECEIVED | id=evt_... | dateCreated=...
[webhook] ✅ Charge updated successfully: payment_id=pay_... | status=RECEIVED
```

## 📊 Eventos Suportados

O webhook processa os seguintes eventos:

| Evento | Descrição |
|--------|-----------|
| `PAYMENT_CREATED` | Cobrança criada |
| `PAYMENT_UPDATED` | Cobrança atualizada |
| `PAYMENT_CONFIRMED` | Pagamento confirmado |
| `PAYMENT_RECEIVED` | Pagamento recebido |
| `PAYMENT_OVERDUE` | Cobrança vencida |
| `PAYMENT_DELETED` | Cobrança deletada |
| `PAYMENT_RESTORED` | Cobrança restaurada |
| `PAYMENT_REFUNDED` | Pagamento estornado |
| `PAYMENT_RECEIVED_IN_CASH` | Pagamento recebido em dinheiro |

**Referência completa**: https://docs.asaas.com/docs/eventos-de-webhooks#eventos-para-cobran%C3%A7as

## 🔍 Debug

### Logs de Webhooks no Asaas

1. Acesse **Menu do Usuário** → **Integrações** → **Logs de Webhooks**
2. Visualize todas as requisições enviadas, status HTTP retornado e payload

### Logs do charges-service

Todos os eventos são logados com prefixo `[webhook]`:

- ✅ Sucesso
- ⚠️  Aviso (evento sem payment object, cobrança não encontrada)
- ❌ Erro (falha ao processar)

## 🚨 Troubleshooting

### Webhook não está sendo chamado

1. Verifique se o webhook está **habilitado** no painel do Asaas
2. Confirme que a **URL está acessível** (teste com curl)
3. Verifique os **logs de webhooks** no painel do Asaas

### Erro 401 (Unauthorized)

- O `asaas-access-token` enviado pelo Asaas não corresponde ao `ASAAS_WEBHOOK_SECRET` configurado no `.env`
- Verifique se você configurou o mesmo token em ambos os lugares

### Erro 400 (Bad Request)

- O JSON enviado pelo Asaas está malformado (raro)
- Verifique os logs do `charges-service` para detalhes

### Erro 500 (Internal Server Error)

- Erro ao atualizar a cobrança no banco de dados
- Verifique:
  - A cobrança existe na tabela `iam.charges`?
  - O Supabase está acessível?
  - As credenciais do Supabase estão corretas?

### Cobrança não encontrada (warning)

Se você receber:

```
[webhook] ⚠️  Charge not found in iam.charges (provider_charge_id=pay_...)
```

Significa que o Asaas enviou um evento para uma cobrança que não existe na sua base. Isso pode acontecer se:
- A cobrança foi criada diretamente no painel do Asaas (não via API)
- A cobrança foi criada antes da integração com o `charges-service`

**Solução**: O webhook ignora cobranças não encontradas. Apenas cobranças criadas via `/v1/asaas/charges` serão atualizadas.

## 🔗 Referências

- [Documentação oficial do Asaas - Webhooks](https://docs.asaas.com/docs/receba-eventos-do-asaas-no-seu-endpoint-de-webhook)
- [Eventos de Webhooks - Cobranças](https://docs.asaas.com/docs/eventos-de-webhooks#eventos-para-cobran%C3%A7as)
- [Como implementar idempotência em Webhooks](https://docs.asaas.com/docs/como-implementar-idempotencia-em-webhooks)

## 📝 Notas Importantes

1. **Idempotência**: O webhook usa `upsert` no banco, então receber o mesmo evento múltiplas vezes é seguro
2. **Resposta Rápida**: O handler retorna `200 OK` imediatamente para evitar timeout na fila do Asaas
3. **IPs do Asaas**: Para produção, considere configurar firewall para aceitar apenas os [IPs oficiais do Asaas](https://docs.asaas.com/docs/ips-oficiais-do-asaas)
