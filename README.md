# go-speeddials

一个用 Go 实现的快速拨号页面，支持目录、拖动排序、图片图标、每日一图背景及备案信息配置。

## 功能特性

- 页面直接编辑快速拨号项，支持图标/图片
- SQLite 存储配置
- 目录分类（单层级，点击弹出目录内容）
- 登录后支持拖动排序、编辑、删除
- 页脚备案信息支持自定义 HTML
- 每日一图（必应每日壁纸）背景
- 支持 Docker / Docker Compose 部署

## 环境变量

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `ADDR` | 监听地址 | `:8080` |
| `DATA_DIR` | 数据目录 | `data` |
| `ADMIN_USER` | 管理员用户名 | `admin` |
| `ADMIN_PASSWORD` | 管理员密码 | `123456` |

## Docker Compose 部署

```bash
docker compose up -d
```

`docker-compose.yml`：

```yaml
services:
  speeddials:
    image: ghcr.io/jiamuliuquan/go-speeddials:latest
    container_name: speeddials
    restart: always
    ports:
      - "8080:8080"
    environment:
      - ADMIN_USER=admin
      - ADMIN_PASSWORD=123456
      - TZ=Asia/Shanghai
    volumes:
      - speeddials-data:/app/data

volumes:
  speeddials-data:
```

常用命令：

```bash
docker compose up -d        # 启动
docker compose down         # 停止
docker compose pull         # 拉取最新镜像
docker compose up -d --force-recreate   # 更新后重建
```

数据保存在 `speeddials-data` 卷中，删除容器不会丢失配置。首次启动后访问 `http://localhost:8080`，右上角登录即可管理。
