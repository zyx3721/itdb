# 项目名称：itdb
# 文件名称：Dockerfile
# 创建时间：2026-03-17 20:14:01
# 系统用户：jerion
# 作　　者：Jerion
# 联系邮箱：416685476@qq.com
# 功能描述：多阶段构建 itdb 镜像，Nginx 托管前端 + Go 后端，基于 alpine 最小化运行环境

# ─── 阶段一：构建前端 ───────────────────────────────────────────
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm ci --registry=https://registry.npmmirror.com

COPY frontend/ ./
RUN npm run build

# ─── 阶段二：构建后端 ───────────────────────────────────────────
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app/backend

# 切换 apk 为阿里云镜像源，安装 CGO 依赖（sqlite 需要）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
 && apk add --no-cache gcc musl-dev

ENV CGO_ENABLED=1
ENV GOPROXY=https://goproxy.cn,direct

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN go build -ldflags="-s -w" -o itdb-server ./main.go

# ─── 阶段三：最终运行镜像 ────────────────────────────────────────
FROM alpine:3.21

WORKDIR /app

ENV TZ=Asia/Shanghai

# 安装 Nginx 和运行时依赖
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
 && apk add --no-cache nginx ca-certificates tzdata && \
    mkdir -p /app/data /app/dist /run/nginx

# 从构建阶段复制产物
COPY --from=backend-builder /app/backend/itdb-server ./
COPY --from=frontend-builder /app/frontend/dist ./dist
COPY nginx.conf /etc/nginx/http.d/default.conf

# 启动脚本：同时启动 Nginx 和 Go 后端
RUN printf '#!/bin/sh\n./itdb-server &\nnginx -g "daemon off;"\n' > /app/start.sh && \
    chmod +x /app/start.sh

EXPOSE 80

ENTRYPOINT ["/app/start.sh"]
