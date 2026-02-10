# Deploy no EasyPanel - Charges Service

## 🔧 Configuração do Serviço

### 1. **Tipo de Aplicação**
- Selecione: **App (Docker Image)**

### 2. **Build Configuration**

#### **Dockerfile Path**
```
Dockerfile
```
⚠️ **IMPORTANTE**: Configure como `Dockerfile` (nome do arquivo), NÃO como caminho de diretório.

#### **Build Context**
```
.
```
Deixe como `.` (diretório raiz do repositório)

### 3. **Environment Variables (Build Args)**

Configure as seguintes variáveis de ambiente:

```bash
# Server
PORT=8083

# Supabase
SUPABASE_URL=https://mzazdlelnaarvfnertjd.supabase.co
SUPABASE_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Im16YXpkbGVsbmFhcnZmbmVydGpkIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImlhdCI6MTc0NjYxODk3NCwiZXhwIjoyMDYyMTk0OTc0fQ.xvTam5Wpfw1sRhqakYWDQ6JXMIbaxm6X1bHObp4dzOM
SUPABASE_SCHEMA=iam

# Webhook
WEBHOOK_URL=https://charges-api.carteiracontabil.com/asaas/feecharges
ASAAS_WEBHOOK_SECRET=lFNVw8q5WMeR3U9FOVOABTp36zrkvtaa

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:8083,http://localhost:4200,https://web.carteiracontabil.com

# Swagger (opcional)
SWAGGER_HOST=charges-api.carteiracontabil.com
SWAGGER_SCHEMES=https
```

### 4. **Port Configuration**
- **Container Port**: `8083`
- **Protocol**: `HTTP`

### 5. **Health Check**
- **Path**: `/health`
- **Interval**: `30s`
- **Timeout**: `5s`
- **Retries**: `3`

### 6. **Domain**
Configure o domínio para apontar para o serviço:
```
charges-api.carteiracontabil.com
```

## 🚀 Deploy Manual (alternativa)

Se o EasyPanel continuar com erro, você pode fazer o build e push manualmente:

### 1. Build da imagem
```bash
cd /home/andrev/Documents/works/carteiracontabil/workspace/charges-service

docker build -t charges-service:latest \
  --build-arg PORT=8083 \
  --build-arg SUPABASE_URL=https://mzazdlelnaarvfnertjd.supabase.co \
  --build-arg SUPABASE_KEY=eyJhbGc... \
  --build-arg SUPABASE_SCHEMA=iam \
  --build-arg WEBHOOK_URL=https://charges-api.carteiracontabil.com/asaas/feecharges \
  --build-arg ASAAS_WEBHOOK_SECRET=lFNVw8q... \
  --build-arg CORS_ALLOWED_ORIGINS=http://localhost:8083,http://localhost:4200,https://web.carteiracontabil.com \
  --build-arg SWAGGER_HOST=charges-api.carteiracontabil.com \
  --build-arg SWAGGER_SCHEMES=https \
  .
```

### 2. Tag para o registry
```bash
docker tag charges-service:latest registry.easypanel.io/charges-service:latest
```

### 3. Push para o registry
```bash
docker push registry.easypanel.io/charges-service:latest
```

## 🐛 Troubleshooting

### Erro: "failed to read dockerfile: open code: no such file or directory"

**Causa**: O campo "Dockerfile Path" está configurado incorretamente.

**Solução**:
1. Vá em **Settings** → **Build**
2. No campo **Dockerfile Path**, coloque apenas: `Dockerfile`
3. No campo **Context**, coloque: `.`
4. Salve e faça o deploy novamente

### Erro: Build timeout

**Solução**:
1. Aumente o timeout de build nas configurações
2. Use cache do Docker se disponível
3. Considere fazer o build localmente e fazer push da imagem

## 📝 Arquivos Importantes

- `Dockerfile` - Configuração do container
- `.dockerignore` - Arquivos ignorados no build
- `go.mod` / `go.sum` - Dependências Go
- `cmd/api/main.go` - Entry point da aplicação

## ✅ Verificação Pós-Deploy

Após o deploy, teste os endpoints:

```bash
# Health check
curl https://charges-api.carteiracontabil.com/health

# Swagger
curl https://charges-api.carteiracontabil.com/swagger/index.html
```
