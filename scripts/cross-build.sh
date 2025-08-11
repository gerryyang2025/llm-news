#!/bin/bash

# 设置颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# 获取当前目录的绝对路径
ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
OUTPUT_DIR="$ROOT_DIR/bin"
APP_NAME="llm-news"

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

echo -e "${YELLOW}开始跨平台构建 LLM News...${NC}"

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: Go 未安装，请先安装 Go 1.21 或更高版本${NC}"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}检测到 Go 版本: ${GO_VERSION}${NC}"

# 支持的平台
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
)

# 构建所有平台
cd "$ROOT_DIR"

# 清理旧文件
echo -e "${YELLOW}清理旧的构建文件...${NC}"
rm -rf "$OUTPUT_DIR"/*

for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}

    if [ "$GOOS" == "windows" ]; then
        OUTPUT_FILE="$OUTPUT_DIR/$APP_NAME-$GOOS-$GOARCH.exe"
    else
        OUTPUT_FILE="$OUTPUT_DIR/$APP_NAME-$GOOS-$GOARCH"
    fi

    echo -e "${YELLOW}构建 $GOOS/$GOARCH...${NC}"

    # 设置构建环境变量
    export CGO_ENABLED=0
    export GOOS=$GOOS
    export GOARCH=$GOARCH

    # 构建
    go build -ldflags="-s -w" -o "$OUTPUT_FILE" ./cmd/server/main.go

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}$GOOS/$GOARCH 构建成功!${NC}"

        # 为类Unix系统设置可执行权限
        if [ "$GOOS" != "windows" ]; then
            chmod +x "$OUTPUT_FILE"
        fi

        # 显示文件大小
        FILE_SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
        echo -e "${GREEN}二进制文件大小: $FILE_SIZE${NC}"
    else
        echo -e "${RED}$GOOS/$GOARCH 构建失败${NC}"
    fi

    echo ""
done

echo -e "${GREEN}所有平台构建完成。构建结果位于: $OUTPUT_DIR${NC}"
ls -lh "$OUTPUT_DIR"