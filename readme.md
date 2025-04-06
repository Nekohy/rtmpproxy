# 说明

本工具是一个基于 Go 语言开发的 RTMP 代理客户端，用于将 RTMP 流通过代理转发到远程 RTMPS 服务器（因为obs不提供内置代理）

## 参数说明

* `-listen`：指定监听的端口，默认为 `10272`。
* `-remote`：指定远程 RTMPS 服务器的地址，例如：`dc5-1.rtmp.t.me:443`。
* `-proxy`：指定 SOCKS5 代理地址，格式为 `socks5://[username:password@]host:port`。例如：`socks5://user:pass@127.0.0.1:7890`,`socks5://127.0.0.1:7890`。
## 使用方法

1.  **启动代理客户端：**

    在命令行中执行以下命令，根据需要替换参数：

    ```bash
    ./rtmp-proxy -listen 10272 -remote dc5-1.rtmp.t.me:443 -proxy socks5://127.0.0.1:7890
    ```

2.  **在 OBS 或其他推流客户端中配置：**

    将推流地址设置为 `rtmp://<listen地址>/<原后缀>`。例如，如果你的监听地址是 `127.0.0.1:10272`，原后缀是 `streamkey`，则推流地址为 `rtmp://127.0.0.1:10272/streamkey`。
