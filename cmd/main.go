package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"sync"
)

type ConnectInfo struct {
	ListenAddr string
	RemoteAddr string
	ProxyAddr  string
	ProxyUser  string
	ProxyPass  string
}

func main() {
	listenAddr := flag.String("listen", ":10721", "Local address to listen on (e.g., :10721)")
	remoteAddr := flag.String("remote", "", "Remote RTMPS server address (e.g., dc5-1.rtmp.t.me:443)")
	proxyAddr := flag.String("proxy", "", "Proxy server address (e.g., socks5://127.0.0.1:7890)")

	flag.Parse()

	if *remoteAddr == "" {
		fmt.Println("Error: -remote flag is required")
		flag.Usage()
		os.Exit(1)
	}

	log.Printf("Starting RTMPS proxy...")
	log.Printf("Listening on: %s", *listenAddr)
	log.Printf("Forwarding to: %s", *remoteAddr)

	// 启动本地监听
	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", *listenAddr, err)
	}
	defer listener.Close()

	log.Println("Waiting for client connections...")
	dialer, err := createDialer(proxyAddr)
	if err != nil {
		log.Fatalf("Failed to create proxy dialer: %v", err)
	}
	// 循环接受客户端连接
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue // 继续接受下一个连接
		}
		// 为每个客户端连接启动一个独立的 goroutine 处理
		go handleClient(clientConn, *remoteAddr, dialer)
	}
}

// createDialer 创建proxy dialer
func createDialer(proxyAddr *string) (proxy.Dialer, error) {
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

// handleClient 主处理函数，协调客户端连接的处理流程
func handleClient(clientConn net.Conn, remoteAddr string, dialer proxy.Dialer) {
	// 始终确保客户端连接在函数退出时关闭
	defer clientConn.Close()
	clientIP := clientConn.RemoteAddr().String()
	log.Printf("[%s] Accepted connection", clientIP)

	// 1. 通过代理连接远程服务器
	proxyConn, err := connectRemoteViaProxy(remoteAddr, dialer, clientIP)
	if err != nil {
		// 连接失败，日志已在 connectRemoteViaProxy 中记录
		return // clientConn 会被顶部的 defer 关闭
	}
	// 注意：此时 proxyConn 已打开，需要确保它最终被关闭。
	// 我们不在这里 defer proxyConn.Close()，而是依赖后续的 tlsConn.Close()
	// 或者在 establishTLS 失败时手动关闭。

	// 2. 在代理连接上建立 TLS 连接
	tlsConn, err := establishTLS(proxyConn, remoteAddr, clientIP)
	if err != nil {
		// TLS 建立失败，日志已在 establishTLS 中记录
		// **重要**: 因为 tlsConn 未成功建立，必须手动关闭 proxyConn
		proxyConn.Close()
		return // clientConn 会被顶部的 defer 关闭
	}
	// TLS 握手成功! defer tlsConn.Close() 会负责关闭 TLS 层和底层的 proxyConn
	defer tlsConn.Close()
	log.Printf("[%s] TLS handshake successful with %s", clientIP, remoteAddr)

	// 3. 双向转发数据
	log.Printf("[%s] Starting data relay between client and %s", clientIP, remoteAddr)
	relayData(clientConn, tlsConn, clientIP) // relayData 内部处理等待逻辑

	log.Printf("[%s] Data relay finished. Closing connections via defer.", clientIP)
}

// connectRemoteViaProxy 通过指定的代理连接到远程地址
func connectRemoteViaProxy(remoteAddr string, dialer proxy.Dialer, clientIP string) (net.Conn, error) {
	log.Printf("[%s] Dialing remote server %s via proxy...", clientIP, remoteAddr)
	proxyConn, err := dialer.Dial("tcp", remoteAddr)
	if err != nil {
		log.Printf("[%s] Failed to dial remote server via proxy: %v", clientIP, err)
		return nil, err // 返回错误，让调用者处理
	}
	log.Printf("[%s] Successfully connected to remote via proxy", clientIP)
	return proxyConn, nil
}

// establishTLS 在给定的原始连接上建立 TLS 客户端连接
func establishTLS(rawConn net.Conn, remoteAddr string, clientIP string) (*tls.Conn, error) {
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
		// 注意：这里不关闭 rawConn，由调用者 (handleClient) 根据情况决定是否关闭
		return nil, err // 返回错误
	}
	// TLS 握手成功，返回 tlsConn
	return tlsConn, nil
}

// relayData 在两个连接之间双向转发数据，并等待转发完成
func relayData(conn1, conn2 net.Conn, clientIP string) {
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
		conn2.Close() // 会让另一个 io.Copy 也很快结束
	}()

	// Goroutine 2: conn2 -> conn1
	go func() {
		defer wg.Done()
		bytesCopied, err := io.Copy(conn1, conn2)
		log.Printf("[%s] Relay %s -> %s finished. Bytes: %d. Error: %v", clientIP, conn2Addr, conn1Addr, bytesCopied, err)
		conn1.Close()
	}()

	// 等待两个 io.Copy goroutine 都完成
	wg.Wait()
}
