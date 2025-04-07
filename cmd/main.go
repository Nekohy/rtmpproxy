package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"rtmpproxy/internal"
)

func main() {
	listenAddr := flag.String("listen", ":10721", "Local address to listen on (e.g., :10721)")
	remoteAddr := flag.String("remote", "", "Remote RTMPS server address (e.g., dc5-1.rtmp.t.me:443)")
	proxyAddr := flag.String("proxy", "", "Socks5 proxy server address (e.g., socks5://username:password@127.0.0.1:7890)")
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
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)

	connection := internal.Connection{
		ProxyAddr:  proxyAddr,
		RemoteAddr: remoteAddr,
	}

	log.Println("Waiting for client connections...")
	dialer, err := connection.CreateDialer()
	if err != nil {
		log.Fatalf("Failed to create proxy dialer: %v", err)
	}
	errChan := make(chan error) // 创建一个通道用于存储错误
	// 启动一个 goroutine 来处理错误
	go func() {
		for err := range errChan {
			if err != nil {
				log.Printf("Error from handler goroutine: %v", err)
			}
		}
	}()

	// 循环接受客户端连接
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue // 继续接受下一个连接
		}

		// 为每个客户端连接启动一个独立的 goroutine 处理
		go func(conn net.Conn) {
			// 在 goroutine 内部调用 HandleClient
			err := connection.HandleClient(conn, dialer)
			// 将错误（可能是 nil）发送到通道
			errChan <- err
		}(clientConn) // 将 clientConn 作为参数传递给 goroutine 的闭包
	}
}
