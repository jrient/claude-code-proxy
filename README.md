# Claude Code Proxy

专注管理和统计分析的 Claude Code 多源代理，支持 Web 管理面板，Docker 一键部署。

## 功能特性

- **多源代理** - 支持 Anthropic 原生和 OpenAI 兼容格式的 API 源
- **协议转换** - 自动将 Anthropic Messages API 请求转换为 OpenAI 格式（反之亦然）
- **模型映射** - 可将请求中的模型名映射到不同 API 源的实际模型名
- **优先级路由** - 按优先级 + 权重的负载均衡策略选择 API 源
- **自动故障转移** - 连续失败自动熔断，健康检查自动恢复
- **多用户管理** - 虚拟 API Key 生成、独立配额和限流
- **统计分析** - 异步收集请求日志，按用户/模型/源多维度统计
- **Web 管理面板** - 中文界面，仪表板、源管理、密钥管理、统计图表
- **单二进制部署** - Go 编译，前端 embed 嵌入，零依赖运行

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go + Gin |
| 前端 | React + Vite + Tailwind + Recharts |
| 数据库 | SQLite (WAL 模式) |
| 部署 | Docker + docker-compose |

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/jrient/claude-code-proxy.git
cd claude-code-proxy
```

### 2. 配置

```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml 配置你的 API 源（也可以在 Web 面板中配置）
```

### 3. 构建

```bash
# 构建前端
cd web && npm install && npm run build && cd ..
cp -r web/dist cmd/server/dist

# 构建后端
CGO_ENABLED=1 go build -o claude-code-proxy ./cmd/server/
```

### 4. Docker 部署

```bash
# 设置管理密码
export ADMIN_PASSWORD=your_password

# 启动
docker compose up -d
```

- 代理端口: `http://localhost:8086`
- 管理面板: `http://localhost:8087`

### 5. 使用

```bash
# 配置 Claude Code 使用代理
export ANTHROPIC_BASE_URL=http://localhost:8086
export ANTHROPIC_API_KEY=ccp-xxxxx  # 在管理面板创建的虚拟 Key
claude
```

## 配置说明

### config.yaml

```yaml
server:
  port: 8080        # 代理端口（容器内部）
  admin_port: 8081  # 管理面板端口（容器内部）

auth:
  admin_password: "changeme"  # 管理面板密码

database:
  path: "./data/proxy.db"

providers:
  - name: "my-provider"
    type: "openai"          # openai 或 anthropic
    base_url: "https://api.example.com/v1"
    api_key: "${API_KEY}"   # 支持环境变量
    priority: 1             # 优先级，数字越小越优先
    weight: 10              # 权重，同优先级按权重负载均衡
    models:                 # 模型映射（可选）
      - source: "claude-sonnet-4-20250514"
        target: "actual-model-name"
```

### Provider 类型

- **anthropic** - Anthropic 原生格式，请求直接转发（模型名可映射）
- **openai** - OpenAI 兼容格式，自动进行请求/响应协议转换

### docker-compose.yml

```yaml
services:
  claude-code-proxy:
    build: .
    ports:
      - "8086:8080"   # 代理端口
      - "8087:8081"   # 管理面板
    volumes:
      - ./data:/app/data
      - ./config.yaml:/app/config.yaml:ro
      - ./claude-code-proxy:/app/claude-code-proxy:ro
      - ./cmd/server/dist:/app/cmd/server/dist:ro
    environment:
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-changeme}
    restart: unless-stopped
```

## 管理面板

访问 `http://localhost:8087`，使用管理密码登录。

- **仪表板** - 总请求数、Token 用量、预估费用、成功率等概览
- **API 源管理** - 添加/编辑/删除 API 源，配置模型映射，查看健康状态
- **API 密钥** - 创建虚拟 Key，设置速率限制和 Token 配额
- **统计分析** - 请求趋势图、模型用量分布、最近请求日志

## API 端点

### 代理端口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /v1/messages | Anthropic Messages API（主要代理端点） |
| GET | /health | 健康检查 |

### 管理端口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/login | 管理员登录 |
| GET | /api/dashboard | 仪表板数据 |
| GET/POST | /api/providers | Provider 列表/创建 |
| PUT/DELETE | /api/providers/:id | Provider 更新/删除 |
| GET/POST | /api/providers/:id/models | 模型映射列表/创建 |
| DELETE | /api/providers/:id/models/:mid | 删除模型映射 |
| GET/POST | /api/apikeys | API Key 列表/创建 |
| PUT/DELETE | /api/apikeys/:id | API Key 更新/删除 |
| GET | /api/stats/timeseries | 时间序列统计 |
| GET | /api/stats/models | 模型统计 |
| GET | /api/stats/logs | 请求日志 |

## License

MIT
