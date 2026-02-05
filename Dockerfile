# 第一阶段：构建前端
FROM node:25.6-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ .
RUN npm run build

# 第二阶段：构建后端
FROM golang:alpine AS backend-builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /app/server
# 安装 CGO (SQLite) 构建依赖
RUN apk add --no-cache gcc musl-dev

COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
# 构建二进制文件
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o minisys-dashboard ./cmd/server

# 第三阶段：最终镜像
FROM alpine:latest
WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata

COPY --from=backend-builder /app/server/minisys-dashboard .
COPY --from=frontend-builder /app/web/dist ./static

EXPOSE 8080
CMD ["./minisys-dashboard"]
