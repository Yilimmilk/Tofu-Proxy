# 使用一个轻量级的基础镜像
FROM alpine

# 将编译好的 Go 程序复制到容器中
COPY main /app/main
COPY index.html /app/index.html

# 设置工作目录
WORKDIR /app

# 设置程序运行时的默认参数
CMD ["./main"]
