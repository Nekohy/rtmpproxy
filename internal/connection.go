package internal

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"log"
	"net"
	"net/url"
	"sync"
)

type Connection struct{}

type ConnectionInterface interface {
	CreateDialer(proxyAddr *string) (proxy.Dialer, error)
	HandleClient(clientConn net.Conn, remoteAddr string, dialer proxy.Dialer) error
	ConnectRemoteAddress(remoteAddr string, dialer proxy.Dialer, clientIP string) (net.Conn, error)
	EstablishTLS(rawConn net.Conn, remoteAddr string, clientIP string) (*tls.Conn, error)
}

// CreateDialer 创建proxy dialer
func (c *Connection) CreateDialer(proxyAddr *string) (proxy.Dialer, error) {
	// 设置代理拨号器 (Dialer)
	var dialer proxy.Dialer

	if *proxyAddr == "" {
		log.Printf("Direct to the remote server")
		dialer = proxy.Direct
	} else {
		var auth *proxy.Auth
		// 使用 url.Parse 解析字符串
		proxyURL, err := url.Parse(*proxyAddr)
		if err != nil {
			return nil, fmt.Errorf("解析代理 URL '%s' 失败: %v", *proxyAddr, err)
		}
		// 提取代理服务器地址 (Host 字段包含地址和端口)
		proxyAddress := proxyURL.Host
		if proxyAddress == "" {
			return nil, fmt.Errorf("invalid proxy URL '%s'", *proxyAddr)
		} else {
			if proxyURL.Scheme != "socks5" {
				return nil, fmt.Errorf("not support proxy scheme %s", proxyURL.Scheme)
			}
			if proxyURL.User != nil {
				username := proxyURL.User.Username()
				password, passwordSet := proxyURL.User.Password() // passwordSet 会告诉你密码是否真的在URL中设置了
				// 只有当用户名或密码实际存在时才创建 auth 结构体
				if username != "" {
					auth = &proxy.Auth{
						User:     username,
						Password: password,
					}
					log.Printf("Proxy Username %s, Proxy Password %s,try to connect with authentication", username, password)
				} else if !passwordSet {
					return nil, fmt.Errorf("invalid authentication in proxy URL")
				} else {
					log.Println("No authentication in proxy URL,try to connect with no authentication")
				}
			}
		}

		dialer, err = proxy.SOCKS5("tcp", proxyAddress, auth, proxy.Direct)
		if err != nil {
			// 创建配置失败报错
			return nil, fmt.Errorf("failed to create proxy dialer: %v", err) // 创建dialer配置失败
		}
		log.Printf("Using proxy: %s", *proxyAddr)
	}
	return dialer, nil
}

// HandleClient 主处理函数，协调客户端连接的处理流程
func (c *Connection) HandleClient(clientConn net.Conn, remoteAddr string, dialer proxy.Dialer) error {
	// 始终确保客户端连接在函数退出时关闭
	defer func(clientConn net.Conn) {
		_ = clientConn.Close()
	}(clientConn)
	clientIP := clientConn.RemoteAddr().String()
	log.Printf("[%s] Accepted connection", clientIP)

	// 1. 通过代理连接远程服务器
	proxyConn, err := c.ConnectRemoteAddress(remoteAddr, dialer, clientIP)
	if err != nil {
		// 连接失败，日志已在 connectRemoteAddress 中记录
		return err // clientConn 会被顶部的 defer 关闭
	}
	// 注意：此时 proxyConn 已打开，需要确保它最终被关闭。
	// 我们不在这里 defer proxyConn.Close()，而是依赖后续的 tlsConn.Close()
	// 或者在 EstablishTLS 失败时手动关闭。

	// 2. 在代理连接上建立 TLS 连接
	tlsConn, err := c.EstablishTLS(proxyConn, remoteAddr, clientIP)
	if err != nil {
		// TLS 建立失败，日志已在 EstablishTLS 中记录
		// **重要**: 因为 tlsConn 未成功建立，必须手动关闭 proxyConn
		_ = proxyConn.Close()
		return err // clientConn 会被顶部的 defer 关闭
	}
	// TLS 握手成功! defer tlsConn.Close() 会负责关闭 TLS 层和底层的 proxyConn
	defer func(tlsConn *tls.Conn) {
		_ = tlsConn.Close()
	}(tlsConn)
	log.Printf("[%s] TLS handshake successful with %s", clientIP, remoteAddr)

	// 3. 双向转发数据
	log.Printf("[%s] Starting data relay between client and %s", clientIP, remoteAddr)
	c.relayData(clientConn, tlsConn, clientIP) // relayData 内部处理等待逻辑

	log.Printf("[%s] Data relay finished. Closing connections via defer.", clientIP)
	return nil
}

// ConnectRemoteAddress 连接到远程地址
func (c *Connection) ConnectRemoteAddress(remoteAddr string, dialer proxy.Dialer, clientIP string) (net.Conn, error) {
	log.Printf("[%s] Dialing remote server %s", clientIP, remoteAddr)
	proxyConn, err := dialer.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("[%s] Failed to dial remote server: %v", clientIP, err)
		return nil, err
	}
	log.Printf("[%s] Successfully connected to remote", clientIP)
	return proxyConn, nil
}

// EstablishTLS 在给定的原始连接上建立 TLS 客户端连接
func (c *Connection) EstablishTLS(rawConn net.Conn, remoteAddr string, clientIP string) (*tls.Conn, error) {
	// 从 remoteAddr 中提取主机名用于 TLS ServerName (SNI)
	remoteHost, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		remoteHost = remoteAddr
		if net.ParseIP(remoteHost) != nil {
			log.Printf("[%s] Warning: Remote address '%s' looks like an IP address. TLS validation might fail or require specific server configuration (SNI).", clientIP, remoteHost)
		}
	}

	// 创建 TLS 配置
	tlsConfig := &tls.Config{
		ServerName:         remoteHost,
		InsecureSkipVerify: false, // !! 重要: 生产环境保持 false !!
	}

	log.Printf("[%s] Starting TLS handshake with %s (ServerName: %s)...", clientIP, remoteAddr, remoteHost)
	// 在 rawConn 之上创建 TLS 客户端连接
	tlsConn := tls.Client(rawConn, tlsConfig)

	// 执行 TLS 握手
	err = tlsConn.Handshake()
	if err != nil {
		log.Printf("[%s] TLS handshake failed with remote %s: %v", clientIP, remoteAddr, err)
		return nil, err // 返回错误
	}
	// TLS 握手成功，返回 tlsConn
	return tlsConn, nil
}

// relayData 在两个连接之间双向转发数据，并等待转发完成
func (c *Connection) relayData(conn1, conn2 net.Conn, clientIP string) {
	var wg sync.WaitGroup
	wg.Add(2) // 等待两个方向的拷贝

	conn1Addr := conn1.RemoteAddr().String()
	conn2Addr := conn2.RemoteAddr().String()

	// Goroutine 1: conn1 -> conn2
	go func() {
		defer wg.Done()
		bytesCopied, err := io.Copy(conn2, conn1)
		log.Printf("[%s] Relay %s -> %s finished. Bytes: %d. Error: %v", clientIP, conn1Addr, conn2Addr, bytesCopied, err)
		// 可以尝试关闭写端，通知对端数据发送完毕，但这依赖具体协议和实现
		// 通常在 TCP 代理中，一方结束后，关闭整个连接更常见和简单
		_ = conn2.Close() // 会让另一个 io.Copy 也很快结束
	}()

	// Goroutine 2: conn2 -> conn1
	go func() {
		defer wg.Done()
		bytesCopied, err := io.Copy(conn1, conn2)
		log.Printf("[%s] Relay %s -> %s finished. Bytes: %d. Error: %v", clientIP, conn2Addr, conn1Addr, bytesCopied, err)
		_ = conn1.Close()
	}()

	// 等待两个 io.Copy goroutine 都完成
	wg.Wait()
}
