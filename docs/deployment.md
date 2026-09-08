# Docker 部署指南

本文档面向本地试运行、单机部署和小规模生产部署。项目由五个容器组成：MySQL、Redis、Go API 服务、Go Worker 服务、Nginx 前端。

## 前置要求

- Docker 24+
- Docker Compose v2+
- 至少 4 GB 可用内存
- 如需真实生成图片/视频，请准备可用的 AI 服务 API Key

## 一键启动

在项目根目录执行：

```bash
cp .env.example .env
docker compose up -d --build
```

启动后访问：

- 前端：http://localhost
- 后端健康检查：http://localhost:8080/healthz
- 后端就绪检查：http://localhost:8080/readyz
- API 文档入口：http://localhost/docs/index.html

默认管理员：

- 账号：`admin`
- 密码：`123456`

## 常用配置

`.env` 中最常改的配置：

| 变量 | 说明 |
| --- | --- |
| `WEB_PORT` | 前端 Nginx 暴露端口，默认 `80` |
| `SERVER_PORT` | 后端暴露端口，默认 `8080` |
| `APP_KEY` | 应用密钥，生产环境必须改为随机字符串 |
| `DB_USERNAME` / `DB_PASSWORD` | MySQL 业务账号 |
| `REDIS_PASSWORD` | Redis 密码 |
| `OPENAI_API_KEY` / `GETGOAPI_API_KEY` | AI 服务密钥 |
| `VIDEO_PROVIDER` | 默认视频生成提供商 |
| `AGNES_API_KEY` / `AGNES_BASE_URL` / `AGNES_MODEL` | Agnes 视频生成（agnes-video-2.5-flash）密钥与地址 |
| `AGNES_LLM_MODEL` / `AGNES_IMAGE_MODEL` | Agnes 生文（agnes-3.0-flash）与生图（agnes-image-2.5-flash）模型名 |

## 数据与文件持久化

Compose 使用命名卷保存数据：

- `mysql-data`：MySQL 数据
- `redis-data`：Redis AOF 数据
- `server-uploads`：上传图片、生成视频等公开资源
- `server-storage`：API 和 Worker 共用的日志等运行文件

备份示例：

```bash
docker compose exec mysql mysqldump -u"$DB_USERNAME" -p"$DB_PASSWORD" "$DB_DATABASE" > spirit_fruit.sql
```

## 更新部署

```bash
git pull
docker compose up -d --build
```

后端启动时会自动执行 GORM AutoMigrate，并在空库中初始化管理员和菜单数据。

## 生产建议

- 修改 `.env` 中的默认密码和 `APP_KEY`。
- 用云厂商安全组或防火墙限制 MySQL、Redis 端口，仅允许必要来源访问。
- 建议在公网入口前增加 HTTPS 反向代理，例如 Caddy、Traefik 或宿主机 Nginx。
- 将 `.env`、数据库备份、上传目录纳入密钥和备份管理，不要提交到 Git。
- 如果图片/视频生成任务较重，可以独立扩容 `worker` 服务，避免影响 API 响应。

## 排障命令

```bash
docker compose ps
docker compose logs -f server
docker compose logs -f worker
docker compose logs -f web
docker compose exec server ffmpeg -version
docker compose exec redis redis-cli -a "$REDIS_PASSWORD" ping
```

如果前端页面能打开但接口失败，优先确认：

- `server` 容器健康检查是否通过。
- `worker` 容器是否正常启动并消费任务。
- `web/nginx.conf` 是否仍代理 `/admin/v1/` 到 `server:8080`。
- 浏览器请求地址是否为同源 `/admin/v1/...`，而不是写死的旧域名。
