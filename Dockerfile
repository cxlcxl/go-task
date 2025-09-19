# 使用轻量级alpine镜像
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 从本地复制已编译的可执行文件
COPY task-executor .

# 复制配置文件（如果需要）
COPY config-docker.yaml config.yaml

# 创建日志目录
RUN mkdir -p /app/logs

# 设置时区
ENV TZ=Asia/Shanghai

# 暴露端口
EXPOSE 9999

# 启动应用
CMD ["./task-executor"]