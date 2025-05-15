package internal

// 建立与目标地址的TCP连接

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/proxy"
	"log"
	"net"
	"net/url"
	"rtmpproxy/utils"
)

// CreateDialer 创建proxy dialer
func (c *Config) CreateDialer() error {
	if *c.ProxyAddr == "" {
		log.Printf("Direct to the remote server")
		c.dialer = proxy.Direct
	} else {
		var auth *proxy.Auth
		// 使用 url.Parse 解析字符串
		proxyURL, err := url.Parse(*c.ProxyAddr)
		if err != nil {
			return utils.InvalidProxy
		}
		// 提取代理服务器地址 (Host 字段包含地址和端口)
		proxyAddress := proxyURL.Host
		if proxyAddress == "" {
			return utils.InvalidProxy
		} else {
			if proxyURL.Scheme != "socks5" {
				return utils.InvalidProxyScheme
			}
			if proxyURL.User != nil {
				user := proxyURL.User.Username()
				pass, ok := proxyURL.User.Password()
				if user == "" || !ok {
					return utils.InvalidProxyAuth
				}
				auth = &proxy.Auth{User: user, Password: pass}
			}
		}

		c.dialer, err = proxy.SOCKS5("tcp", proxyAddress, auth, proxy.Direct)
		if err != nil {
			// 创建配置失败报错
			return utils.FailedToCreateProxyDialer // 创建dialer配置失败
		}
		log.Printf("Using proxy: %s", *c.ProxyAddr)
	}
	return nil
}

// EstablishTLS 在给定的原始连接上建立 TLS 客户端连接
func (c *Config) EstablishTLS() error {
	// 从 remoteAddr 中提取主机名用于 TLS ServerName (SNI)
	remoteHost, _, err := net.SplitHostPort(c.RemoteURL.Host)
	if err != nil {
		if net.ParseIP(c.RemoteURL.Host) != nil {
			remoteHost = c.RemoteURL.Host
			log.Printf("[%s] Warning: Remote address looks like an IP address. TLS validation might fail or require specific server configuration (SNI).", remoteHost)
		}
	}

	// 创建 TLS 配置
	tlsConfig := &tls.Config{
		ServerName:         remoteHost,
		InsecureSkipVerify: c.InsecureSkipVerify,
	}

	log.Printf("Starting TLS handshake with %s (ServerName: %s)...", c.RemoteURL.Host, remoteHost)
	// 在 rawConn 之上创建 TLS 客户端连接
	c.conn = tls.Client(c.conn, tlsConfig)

	// 执行 TLS 握手

	err = c.conn.(*tls.Conn).Handshake()
	if err != nil {
		log.Printf("TLS handshake failed with remote %s: %v", c.RemoteURL.Host, err)
		return utils.FailedToEstablishTLS // 返回错误
	}
	// TLS 握手成功，返回 tlsConn
	return nil
}

// ConnectRemoteAddress 连接到远程地址
func (c *Config) ConnectRemoteAddress() (net.Conn, error) {
	var err error
	var useTLS bool
	c.RemoteURL, useTLS, err = utils.ParseLink(*c.RemoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote address '%s': %w", *c.RemoteAddr, err)
	}
	err = c.CreateDialer()
	if err != nil {
		return nil, err
	}

	log.Printf("Dialing remote server %s", c.RemoteURL.Host)
	c.conn, err = c.dialer.Dial("tcp", c.RemoteURL.Host)
	if err != nil {
		log.Printf("Dial error: %v", err)
		return nil, utils.FailedToConnectRemoteServer
	}

	log.Printf("Resolved remote target: %s (TLS: %v)", c.RemoteURL.Host, map[bool]string{true: "TLS", false: "No TLS"}[useTLS])

	if useTLS {
		log.Printf("Establishing TLS connection with %s...", c.RemoteURL.Host)
		// EstablishTLS 应该接收原始连接，返回 TLS 连接
		// 注意：传入 RemoteURL.Hostname() 可能更适合 TLS 验证，而不是 remoteHost (包含端口)
		err = c.EstablishTLS()
		if err != nil {
			return nil, utils.FailedToEstablishTLS
		}
		log.Printf("TLS handshake successful with %s", c.RemoteURL.Host)
	}

	log.Printf("Successfully connected to remote Server")
	return c.conn, nil
}
