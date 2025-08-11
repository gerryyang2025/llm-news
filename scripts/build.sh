#!/bin/bash

# 设置颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# 获取当前目录的绝对路径
ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
OUTPUT_DIR="$ROOT_DIR/bin"
OUTPUT_FILE="$OUTPUT_DIR/llm-news"

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

echo -e "${YELLOW}开始构建 LLM News...${NC}"

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: Go 未安装，请先安装 Go 1.21 或更高版本${NC}"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}检测到 Go 版本: ${GO_VERSION}${NC}"

# 构建参数设置
export CGO_ENABLED=0
export GOOS=$(go env GOOS)
export GOARCH=$(go env GOARCH)

echo -e "${YELLOW}构建目标: $GOOS/$GOARCH${NC}"
echo -e "${YELLOW}构建输出: $OUTPUT_FILE${NC}"

# 删除旧的构建文件（如果存在）
if [ -f "$OUTPUT_FILE" ]; then
    rm "$OUTPUT_FILE"
    echo -e "${YELLOW}已删除旧的构建文件${NC}"
fi

# 开始构建
echo -e "${YELLOW}正在编译...${NC}"
cd "$ROOT_DIR"

# 使用-ldflags设置编译参数，例如去除调试信息，设置版本号等
# -s: 去除符号表
# -w: 去除DWARF调试信息，减小二进制文件大小
# 可以根据需要调整
go build -ldflags="-s -w" -o "$OUTPUT_FILE" ./cmd/server/main.go

# 检查构建结果
if [ $? -eq 0 ] && [ -f "$OUTPUT_FILE" ]; then
    echo -e "${GREEN}构建成功!${NC}"

    # 显示文件大小
    FILE_SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
    echo -e "${GREEN}生成的二进制文件大小: $FILE_SIZE${NC}"

    # 设置可执行权限
    chmod +x "$OUTPUT_FILE"
    echo -e "${GREEN}已设置可执行权限${NC}"

    echo -e "${YELLOW}你可以使用以下命令运行:${NC}"
    echo -e "${GREEN}$OUTPUT_FILE${NC}"
else
    echo -e "${RED}构建失败${NC}"
    exit 1
fi