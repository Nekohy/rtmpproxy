package internal

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultRTMPSPort = "443"
	defaultRTMPPort  = "1935"
	schemeRTMPS      = "rtmps"
	schemeRTMP       = "rtmp"
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
			return nil, InvalidProxy
		}
		// 提取代理服务器地址 (Host 字段包含地址和端口)
		proxyAddress := proxyURL.Host
		if proxyAddress == "" {
			return nil, InvalidProxy
		} else {
			if proxyURL.Scheme != "socks5" {
				return nil, InvalidProxyScheme
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
					return nil, InvaildProxyPassword
				} else {
					log.Println("No authentication in proxy URL,try to connect with no authentication")
				}
			}
		}

		dialer, err = proxy.SOCKS5("tcp", proxyAddress, auth, proxy.Direct)
		if err != nil {
			// 创建配置失败报错
			return nil, FailedToCreateProxyDialer // 创建dialer配置失败
		}
		log.Printf("Using proxy: %s", *proxyAddr)
	}
	return dialer, nil
}

// HandleClient 主处理函数，协调客户端连接的处理流程
func (c *Connection) HandleClient(clientConn net.Conn, remoteAddr string, dialer proxy.Dialer) error {
	// 始终确保客户端连接在函数退出时关闭
	defer func() {
		_ = clientConn.Close()
		log.Printf("[%s] Client connection closed.", clientConn.RemoteAddr().String())
	}()

	clientIP := clientConn.RemoteAddr().String()
	log.Printf("[%s] Accepted connection. Target: %s", clientIP, remoteAddr)

	// 1. 解析远程地址并确定连接参数
	remoteURL, err := url.Parse(remoteAddr)
	if err != nil {
		return fmt.Errorf("[%s] failed to parse remote address '%s': %w", clientIP, remoteAddr, err)
	}

	var remoteHost string // host:port 格式
	var useTLS bool

	switch strings.ToLower(remoteURL.Scheme) {
	case schemeRTMPS:
		useTLS = true
		remoteHost = remoteURL.Host
		if remoteURL.Port() == "" {
			remoteHost = net.JoinHostPort(remoteURL.Hostname(), defaultRTMPSPort)
		}
	case schemeRTMP:
		useTLS = false
		remoteHost = remoteURL.Host
		if remoteURL.Port() == "" {
			remoteHost = net.JoinHostPort(remoteURL.Hostname(), defaultRTMPPort)
		}
	default:
		return UnSupportedScheme
	}

	log.Printf("[%s] Resolved remote target: %s (TLS: %v)", clientIP, remoteHost, useTLS)

	// 连接远程服务器 (建立原始连接)
	rawConn, err := c.ConnectRemoteAddress(remoteHost, dialer, clientIP)
	if err != nil {
		// ConnectRemoteAddress 内部应记录详细错误
		return FailedToConnectRemoteServer // clientConn 会被顶部的 defer 关闭
	}

	// 使用一个变量代表最终与远程服务器交互的连接
	// 并使用 defer 确保其最终关闭 (无论是原始连接还是 TLS 连接)
	var remoteConn = rawConn // 初始设为原始连接
	defer func() {
		if remoteConn != nil {
			log.Printf("[%s] Closing connection to remote %s.", clientIP, remoteHost)
			_ = remoteConn.Close()
		}
	}()

	// 3. 如果需要，建立 TLS 连接
	if useTLS {
		log.Printf("[%s] Establishing TLS connection with %s...", clientIP, remoteHost)
		// EstablishTLS 应该接收原始连接，返回 TLS 连接
		// 注意：传入 remoteURL.Hostname() 可能更适合 TLS 验证，而不是 remoteHost (包含端口)
		tlsConn, tlsErr := c.EstablishTLS(rawConn, remoteURL.Hostname(), clientIP)
		if tlsErr != nil {
			// EstablishTLS 内部应记录详细错误
			// 不需要手动关闭 rawConn，因为 remoteConn 仍然是 rawConn，会被 defer 关闭
			return FailedToEstablishTLS
		}
		// TLS 握手成功! 更新 remoteConn 为 TLS 连接
		// 关闭 tlsConn 时，它会自动关闭底层的 rawConn
		remoteConn = tlsConn
		log.Printf("[%s] TLS handshake successful with %s", clientIP, remoteHost)
	}

	// 4. 双向转发数据
	log.Printf("[%s] Starting data relay between client and %s (%s)", clientIP, remoteHost, map[bool]string{true: "TLS", false: "No TLS"}[useTLS])
	errs := c.relayData(clientConn, remoteConn, clientIP) // relayData 内部处理等待和错误
	for err := range errs {
		if &err != nil { //todo 在这里加入自定义重试处理
			log.Printf("[%s] Connection error: %v", clientIP, err)
		}
	}

	log.Printf("[%s] Data relay finished between client and %s.", clientIP, remoteHost)

	// 连接由 defer 语句负责关闭
	return nil
}

// ConnectRemoteAddress 连接到远程地址
func (c *Connection) ConnectRemoteAddress(remoteAddr string, dialer proxy.Dialer, clientIP string) (net.Conn, error) {
	log.Printf("[%s] Dialing remote server %s", clientIP, remoteAddr)
	proxyConn, err := dialer.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("[%s] Failed to dial remote server: %v", clientIP, err)
		return nil, FailedToConnectRemoteServer
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
		return nil, FailedToEstablishTLS // 返回错误
	}
	// TLS 握手成功，返回 tlsConn
	return tlsConn, nil
}

// relayData 在两个连接之间双向转发数据，并等待转发完成
func (c *Connection) relayData(conn1, conn2 net.Conn, clientIP string) []*error {
	var wg sync.WaitGroup
	errChan := make(chan *error) // 容量为2，确保发送不阻塞
	wg.Add(2)                    // 等待两个方向的拷贝

	conn1Addr := conn1.RemoteAddr().String()
	conn2Addr := conn2.RemoteAddr().String()

	// Goroutine 1: conn1 -> conn2
	go func() {
		defer wg.Done()
		bytesCopied, err := io.Copy(conn2, conn1)
		log.Printf("[%s] Relay %s -> %s finished. Bytes: %d. Error: %v", clientIP, conn1Addr, conn2Addr, bytesCopied, err)
		if err != nil {
			errChan <- &err
		} else {
			errChan <- nil // 正常完成
		}
		_ = conn2.Close() // 会让另一个 io.Copy 也很快结束
	}()

	// Goroutine 2: conn2 -> conn1
	go func() {
		defer wg.Done()
		bytesCopied, err := io.Copy(conn1, conn2)
		log.Printf("[%s] Relay %s -> %s finished. Bytes: %d. Error: %v", clientIP, conn2Addr, conn1Addr, bytesCopied, err)
		if err != nil {
			errChan <- &err
		} else {
			errChan <- nil // 正常完成
		}
		_ = conn1.Close()
	}()

	// 等待两个 io.Copy goroutine 都完成
	wg.Wait()
	close(errChan)

	errs := make([]*error, 0) // 关闭 channel，以便 range 退出
	for err := range errChan {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
