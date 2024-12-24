# 构建阶段
FROM golang:1.22-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o ip-geo cmd/server/main.go

# 运行阶段
FROM alpine:latest

# 安装基本工具
RUN apk --no-cache add ca-certificates tzdata

# 设置时区为上海
ENV TZ=Asia/Shanghai

# 创建必要的目录
RUN mkdir -p /app/logs /app/mmdb

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/ip-geo .
COPY --from=builder /app/config.json .

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./ip-geo"] 