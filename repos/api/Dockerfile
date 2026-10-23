# --- Etapa 1: compilación ---
FROM golang:1.26.10-alpine AS builder

WORKDIR /app

# Instalar dependencias de compilación y certificados
RUN apk add --no-cache git ca-certificates tzdata

# Copiar dependencias de Go para aprovechar cache de Docker
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar binario estático optimizado
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o servidor .

# --- Etapa 2: imagen final de producción ---
FROM alpine:3.22

WORKDIR /app

# Instalar certificados SSL y zona horaria
RUN apk add --no-cache ca-certificates tzdata

# Copiar el binario compilado y las migraciones SQL
COPY --from=builder /app/servidor .
COPY --from=builder /app/migrations ./migrations

# Variables de entorno por defecto
ENV PORT=8080
ENV APP_ENV=production

EXPOSE 8080

CMD ["./servidor"]
