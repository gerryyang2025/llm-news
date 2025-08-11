#!/bin/bash

# 获取当前目录的绝对路径
ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
PID_FILE=$ROOT_DIR/llm-news.pid

# 检查PID文件是否存在
if [ ! -f "$PID_FILE" ]; then
    echo "LLM News service is not running (PID file not found)."
    exit 1
fi

# 读取PID
PID=$(cat $PID_FILE)

# 检查进程是否存在
if ps -p $PID > /dev/null; then
    echo "LLM News service is running with PID: $PID"
    echo "Access URL: http://localhost:8081"

    # 显示服务运行时间
    PROC_STAT=$(ps -o etime= -p $PID)
    echo "Running time: $PROC_STAT"

    # 显示日志文件最后10行
    LOG_FILE=$ROOT_DIR/logs/llm-news.log
    if [ -f "$LOG_FILE" ]; then
        echo ""
        echo "Last 10 lines from log file:"
        tail -n 10 $LOG_FILE
    fi
else
    echo "LLM News service is not running (PID $PID not found)."
    rm -f $PID_FILE
    exit 1
fi