#!/bin/bash

# 创建logs目录（如果不存在）
mkdir -p logs

# 获取当前目录的绝对路径
ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
BINARY_PATH="$ROOT_DIR/bin/llm-news"
NATIVE_BINARY_PATH="$ROOT_DIR/bin/llm-news-$(go env GOOS)-$(go env GOARCH)"
NATIVE_BINARY_PATH_EXE="$ROOT_DIR/bin/llm-news-$(go env GOOS)-$(go env GOARCH).exe"

# 获取本机IP地址
get_local_ip() {
    if [ "$(uname)" == "Darwin" ]; then
        # macOS
        ipconfig getifaddr en0 || ipconfig getifaddr en1 || echo "0.0.0.0"
    else
        # Linux
        ip -4 addr show | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | grep -v '127.0.0.1' | head -n 1 || echo "0.0.0.0"
    fi
}

LOCAL_IP=$(get_local_ip)

# 打印启动信息
echo "Starting LLM News service..."

# 优先使用编译好的二进制文件
if [ -f "$BINARY_PATH" ]; then
    echo "使用已构建的二进制文件: $BINARY_PATH"
    nohup "$BINARY_PATH" > "$ROOT_DIR/logs/llm-news.log" 2>&1 &
elif [ -f "$NATIVE_BINARY_PATH" ]; then
    echo "使用平台特定的二进制文件: $NATIVE_BINARY_PATH"
    nohup "$NATIVE_BINARY_PATH" > "$ROOT_DIR/logs/llm-news.log" 2>&1 &
elif [ -f "$NATIVE_BINARY_PATH_EXE" ]; then
    echo "使用平台特定的二进制文件: $NATIVE_BINARY_PATH_EXE"
    nohup "$NATIVE_BINARY_PATH_EXE" > "$ROOT_DIR/logs/llm-news.log" 2>&1 &
else
    echo "未找到构建好的二进制文件，使用 'go run' 命令运行..."
    # 运行服务，将输出重定向到日志文件，并将进程放入后台
    cd "$ROOT_DIR" && nohup go run cmd/server/main.go > "$ROOT_DIR/logs/llm-news.log" 2>&1 &
fi

# 保存进程ID到文件
echo $! > "$ROOT_DIR/llm-news.pid"

echo "LLM News service started with PID: $(cat $ROOT_DIR/llm-news.pid)"
echo "Log file is located at: $ROOT_DIR/logs/llm-news.log"
echo "You can access the service at: http://$LOCAL_IP:8081"