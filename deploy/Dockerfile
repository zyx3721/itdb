############################
# 1) frontend builder
############################
FROM node:22-alpine AS frontend-builder

WORKDIR /src/frontend

COPY frontend/package*.json ./
RUN npm ci --registry=https://registry.npmmirror.com

COPY frontend/ ./
RUN npm run build

############################
# 2) backend builder
############################
FROM golang:1.25-alpine AS backend-builder

WORKDIR /src/backend

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

ARG CGO_ENABLED=0
ENV CGO_ENABLED=${CGO_ENABLED}
RUN go build -o itdb-backend ./main.go

############################
# 3) runtime
############################
FROM alpine:3.22 AS runtime

ENV TZ=Asia/Shanghai

ARG ALPINE_MIRROR=
RUN if [ -n "$ALPINE_MIRROR" ]; then \
      sed -i "s/dl-cdn.alpinelinux.org/${ALPINE_MIRROR}/g" /etc/apk/repositories; \
    fi \
 && apk add --no-cache nginx supervisor ca-certificates tzdata curl \
 && rm -f /etc/nginx/http.d/default.conf

WORKDIR /app

COPY --from=backend-builder /src/backend/itdb-backend /app/itdb-backend
COPY --from=frontend-builder /src/frontend/dist /usr/share/nginx/html
COPY deploy/nginx.conf /etc/nginx/http.d/default.conf
COPY deploy/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY deploy/entrypoint.sh /entrypoint.sh

RUN sed -i 's/\r$//' /entrypoint.sh && chmod +x /entrypoint.sh \
 && mkdir -p /app/data/logs \
 && ln -s /app/data/logs /app/logs

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD curl -fs http://127.0.0.1/health || exit 1

ENTRYPOINT ["/entrypoint.sh"]