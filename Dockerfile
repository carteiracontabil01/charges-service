FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/charges-service ./cmd/api

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /bin/charges-service /app/charges-service

EXPOSE 8083
ENV PORT=8083

CMD ["/app/charges-service"]

