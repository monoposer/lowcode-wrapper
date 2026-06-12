# dataspan

轻量级 sidecar：通过统一的 PostgREST 风格 REST 接口访问多种异构数据源。

在 PostgreSQL 元数据库或声明式 YAML 中注册 Foreign Server / Table 后，即可通过 `/v1/{schema}/{table}` CRUD 与 `/v1/rpc/{fn}` 访问远程 HTTP、Postgres、MySQL、MongoDB、Redis、S3 或本地文件。

仓库地址：[github.com/monoposer/dataspan](https://github.com/monoposer/dataspan)

## 快速启动

```bash
cp .env.example .env
# 设置 WRAPPER_MASTER_KEY: openssl rand -base64 32
make postgres-up
make migrate
make run
```

- Playground：http://localhost:3020/playground/
- Swagger：http://localhost:3020/swagger/
- 健康检查：`GET /health` 返回 `{"status":"ok","version":"..."}`

## 部署

生产环境部署示例见 **[deploy/](deploy/README.md)** 目录，所有 Docker 相关文件均在此：

- `Dockerfile` / `Dockerfile.cli` — 镜像构建
- `docker-compose.yml` — 本地/部署编排（源码构建 + postgres）
- 环境变量与密钥配置说明
- postgres / file 两种元数据存储模式指引

## Docker 镜像

| 镜像 | 说明 |
|------|------|
| `monoposer/dataspan` | HTTP 服务（server） |
| `monoposer/dataspan-cli` | 元数据转换 CLI（`convert`） |

本地构建：

```bash
make docker-build IMAGE=monoposer/dataspan:latest
make docker-build-cli CLI_IMAGE=monoposer/dataspan-cli:latest
```

推送 `v*` 标签后会自动发布到 Docker Hub。

## Admin API

- `POST /api/credentials` — 存储加密凭据
- `POST /api/servers` — 注册 Foreign Server
- `POST /api/tables` — 注册表与列映射
- `POST /api/functions` — 注册 RPC

## 数据 API（PostgREST 风格）

- `GET /v1/{schema}/{table}?col=eq.val&select=a,b&limit=10`
- `POST /v1/{schema}/{table}`
- `PATCH /v1/{schema}/{table}?id=eq.1`
- `DELETE /v1/{schema}/{table}?id=eq.1`
- `POST /v1/rpc/{function}`

## 文档

| 文档 | 说明 |
|------|------|
| [docs/](docs/README.md) | 架构、store、drivers、模块速查（英文） |
| [deploy/](deploy/README.md) | 部署说明 |
| [.cursor/DEVELOPMENT.md](.cursor/DEVELOPMENT.md) | 本地 env、curl、make |
| [README.md](README.md) | English readme |
| [AGENTS.md](AGENTS.md) | Cursor Agent 入口 |

## 版本

当前版本见 [VERSION](VERSION)。

本地生成/更新版本：

```bash
make version              # 查看当前版本
make version-next         # 预览下次 CI 发布的版本
make version-bump         # patch +1 并写入 VERSION
make version-bump BUMP=minor
make version-set VER=1.0.0
```

合并到 `main` 后会自动 patch 升版、更新 `VERSION` 并推送 `v*` 标签，进而触发：

- GitHub Release（`dataspan-convert` 二进制）
- Docker Hub 镜像 `monoposer/dataspan`、`monoposer/dataspan-cli`

需在仓库 Secrets 中配置 `DOCKERHUB_USERNAME`、`DOCKERHUB_TOKEN`。

## License

[MIT](LICENSE)
