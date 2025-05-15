package internal

import (
	"golang.org/x/net/proxy"
	"net"
	"net/url"
)

type Config struct {
	ListenAddr         *string
	RemoteAddr         *string
	ProxyAddr          *string
	Plugin             *Plugin // 插件功能
	InsecureSkipVerify bool
	RemoteURL          *url.URL     // 解析后的远程地址
	dialer             proxy.Dialer // 内部使用的dialer
	conn               net.Conn     // 连接实例
}

type Plugin struct {
	Name   *string
	Config interface{}
}
