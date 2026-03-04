FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.2

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN swag init -g cmd/main.go -o docs --parseInternal --parseDependency
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /service ./cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl
WORKDIR /app
COPY --from=builder /service .

HEALTHCHECK --interval=10s --timeout=3s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8083/health || exit 1

EXPOSE 8083
CMD ["./service"]
