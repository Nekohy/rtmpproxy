# 说明

一个基于 Go 语言开发的 RTMP Relay Client

## 参数说明

* `-listen`：指定监听的端口，默认为 `1935`。
* `-remote`：指定远程 RTMPS 服务器的地址，例如：`rtmps://dc5-1.rtmp.t.me/s/aaaa`。
* `-proxy`：指定 SOCKS5 代理地址，格式为 `socks5://[username:password@]host:port`。例如：`socks5://user:pass@127.0.0.1:7890`,`socks5://127.0.0.1:7890`
* `-plugin`：指定插件配置，格式为 `pluginName:{"key":"value"}`。例如：`test:{"message":"hello world"}`
* `-ignore`：忽略 TLS 证书验证,默认为 `false`
* `-force`: 强制处理所有数据包，也许对性能有轻微影响，只在必要时启用，可以处理到视频流后的RTMP指令(FCUnpublish的streamName)，默认为 `false`
* `-flashVer`: RTMP的Connect命令使用的flashVer，默认透传原始参数
* `-type`: RTMP的Connect命令使用的type，默认透传原始参数

# 特性
* Pure Golang 实现
* 支持 RTMP Over SockS5
* 支持远程RTMPS服务器
* 支持插件功能
* 修改RTMP Header为原RTMP连接参数

# 使用
按照如上配置参数运行程序，连接 rtmp://127.0.0.1:1935 即可

# 自编译
只需要直接编译即可，如果需要插件请添加对应插件的TAG

# 插件开发
目前已支持Bilibili的自动上下播

开发插件时，可以参考Plugin/test的插件实现

# 感谢
* [vizee/rtmpproxy](https://github.com/vizee/rtmpproxy)