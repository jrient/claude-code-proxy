# Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# Build backend
FROM golang:1.23-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./cmd/server/dist
RUN CGO_ENABLED=1 go build -o claude-code-proxy ./cmd/server

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /app/claude-code-proxy .
COPY config.example.yaml ./config.yaml

EXPOSE 8080 8081
VOLUME /app/data

CMD ["./claude-code-proxy"]
