English | [简体中文](./README.md)

# Claude Code Proxy

A multi-upstream proxy for Claude Code focused on management and usage analytics, with a web admin panel and one-command Docker deployment.

## Features

- **Multi-upstream proxy** - Supports both Anthropic-native and OpenAI-compatible API providers
- **Protocol conversion** - Automatically converts Anthropic Messages API requests to the OpenAI format (and vice versa)
- **Model mapping** - Map model names in requests to the actual model names of different API providers
- **Priority-based routing** - Selects API providers using a priority + weight load-balancing strategy
- **Automatic failover** - Circuit breaking after consecutive failures, with automatic recovery via health checks
- **Multi-user management** - Virtual API key generation with per-key quotas and rate limits
- **Usage analytics** - Asynchronous request-log collection, with statistics broken down by user, model, and provider
- **Web admin panel** - Dashboard, provider management, key management, and statistics charts (Chinese UI)
- **Single-binary deployment** - Compiled with Go, frontend embedded via `embed`, zero runtime dependencies

## Tech Stack

| Layer | Technology |
|------|------|
| Backend | Go + Gin |
| Frontend | React + Vite + Tailwind + Recharts |
| Database | SQLite (WAL mode) |
| Deployment | Docker + docker-compose |

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/jrient/claude-code-proxy.git
cd claude-code-proxy
```

### 2. Configure

```bash
cp config.example.yaml config.yaml
# Edit config.yaml to set up your API providers (can also be configured in the web panel)
```

### 3. Build

```bash
# Build the frontend
cd web && npm install && npm run build && cd ..
cp -r web/dist cmd/server/dist

# Build the backend
CGO_ENABLED=1 go build -o claude-code-proxy ./cmd/server/
```

### 4. Deploy with Docker

```bash
# Set the admin password
export ADMIN_PASSWORD=your_password

# Start
docker compose up -d
```

- Proxy port: `http://localhost:8086`
- Admin panel: `http://localhost:8087`

### 5. Use

```bash
# Configure Claude Code to use the proxy
export ANTHROPIC_BASE_URL=http://localhost:8086
export ANTHROPIC_API_KEY=ccp-xxxxx  # Virtual key created in the admin panel
claude
```

## Configuration

### config.yaml

```yaml
server:
  port: 8080        # Proxy port (inside the container)
  admin_port: 8081  # Admin panel port (inside the container)

auth:
  admin_password: "changeme"  # Admin panel password

database:
  path: "./data/proxy.db"

providers:
  - name: "my-provider"
    type: "openai"          # openai or anthropic
    base_url: "https://api.example.com/v1"
    api_key: "${API_KEY}"   # Environment variables supported
    priority: 1             # Priority; lower numbers take precedence
    weight: 10              # Weight; load-balanced by weight within the same priority
    models:                 # Model mapping (optional)
      - source: "claude-sonnet-4-20250514"
        target: "actual-model-name"
```

### Provider types

- **anthropic** - Anthropic-native format; requests are forwarded as-is (model names can be mapped)
- **openai** - OpenAI-compatible format; request/response protocol conversion is performed automatically

### docker-compose.yml

```yaml
services:
  claude-code-proxy:
    build: .
    ports:
      - "8086:8080"   # Proxy port
      - "8087:8081"   # Admin panel
    volumes:
      - ./data:/app/data
      - ./config.yaml:/app/config.yaml:ro
      - ./claude-code-proxy:/app/claude-code-proxy:ro
      - ./cmd/server/dist:/app/cmd/server/dist:ro
    environment:
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-changeme}
    restart: unless-stopped
```

## Admin Panel

Visit `http://localhost:8087` and log in with the admin password.

- **Dashboard** - Overview of total requests, token usage, estimated cost, success rate, and more
- **Provider Management** - Add/edit/delete API providers, configure model mappings, view health status
- **API Keys** - Create virtual keys, set rate limits and token quotas
- **Analytics** - Request trend charts, model usage distribution, recent request logs

## API Endpoints

### Proxy port

| Method | Path | Description |
|------|------|------|
| POST | /v1/messages | Anthropic Messages API (main proxy endpoint) |
| GET | /health | Health check |

### Admin port

| Method | Path | Description |
|------|------|------|
| POST | /api/login | Admin login |
| GET | /api/dashboard | Dashboard data |
| GET/POST | /api/providers | List/create providers |
| PUT/DELETE | /api/providers/:id | Update/delete a provider |
| GET/POST | /api/providers/:id/models | List/create model mappings |
| DELETE | /api/providers/:id/models/:mid | Delete a model mapping |
| GET/POST | /api/apikeys | List/create API keys |
| PUT/DELETE | /api/apikeys/:id | Update/delete an API key |
| GET | /api/stats/timeseries | Time-series statistics |
| GET | /api/stats/models | Model statistics |
| GET | /api/stats/logs | Request logs |

## License

MIT
