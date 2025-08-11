FROM golang:1.21-alpine

WORKDIR /app

# 复制Go模块定义文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 设置构建参数
ENV CGO_ENABLED=0
ENV GOOS=linux

# 构建应用
RUN go build -o llm-news ./cmd/server/main.go

# 使用更小的基础镜像
FROM alpine:latest

# 安装ca-certificates以支持HTTPS请求
RUN apk --no-cache add ca-certificates

WORKDIR /app

# 从构建阶段复制编译好的应用
COPY --from=0 /app/llm-news .
COPY --from=0 /app/web ./web

# 暴露应用端口
EXPOSE 8081

# 运行应用
CMD ["./llm-news"]