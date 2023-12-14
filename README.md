# GO-TOFU-PROXY

- 基于 Go 实现的 API HTTP 代理
- 本项目参考了[openai-proxy@geekr-dev](https://github.com/geekr-dev/openai-proxy)

## 功能

- 代理到OpenAI的API。
- 代理到Cloudflare的API。
- 可选地代理到任意指定的站点。
- 返回根路径下的 `index.html` 文件内容。

## 如何使用

1. **准备工作**：
   
    确保您的系统已安装Go环境。

2. **运行程序**：

    ```bash
    go run main.go -port=9000 -enable-proxy-any-site=false
    ```

    参数说明：
    
    - `port`：程序的监听的端口号。（默认为 `9000`，可省略）
    - `enable-proxy-any-site`：是否启用代理任意站点的功能（默认为 `false`，可省略）。

3. **编译打包**：

    ```bash
    ./build.sh
    ```

4. **Docker部署**：

    编译完成后，请使用以下指令即可在Docker内运行：(请先安装Docker和docker-compose)

    ```bash
    docker-compose up -d
    ```

    > NOTE: 请自行根据需要修改docker-compose.yml文件

5. **访问代理**：

    通过以下URL访问代理服务：

    - OpenAI API代理：`http://127.0.0.1:9000/o/...`
    - Cloudflare API代理：`http://127.0.0.1:9000/c/...`
    - 根路径：`http://127.0.0.1:9000/` - 返回 `index.html` 的内容。

    如果启用了代理任意站点的功能，可以通过 `/p` 路径代理到任意站点。

    - 只需要在发起发起代理请求的时候通过 `X-Target-Host` 设置你想要代理的域名（不带 `http(s)://` 前缀）即可，

> ### 关于流式响应SSE
> 如果你是通过 Nginx 这种反向代理对外提供服务，记得通过如下配置项将默认缓冲和缓存关掉才会生效：
>
> ```
> proxy_buffering off;
> proxy_cache off;
> ```

## 注意事项

- 该程序用于学习和测试目的。
- `enable-proxy-any-site`代理任意站点功能可能会导致一些安全问题，请考虑再三再决定是否打开
