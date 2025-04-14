package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"rtmpproxy/internal"
	"rtmpproxy/plugins"
	_ "rtmpproxy/plugins/bilibili"
	"strings"
)

func main() {
	listenAddr := flag.String("listen", ":10721", "Local address to listen on (e.g., :10721)")
	remoteAddr := flag.String("remote", "", "Remote RTMPS server address (e.g., dc5-1.rtmp.t.me:443)")
	proxyAddr := flag.String("proxy", "", "Socks5 proxy server address (e.g., socks5://username:password@127.0.0.1:7890)")
	pluginConfig := flag.String("plugin", "", `Plugin config e.g. "bilibili:{\"参数\":\"value\"}"`)
	flag.Parse()

	baseCfg := &internal.Config{
		ListenAddr: listenAddr,
		RemoteAddr: remoteAddr,
		ProxyAddr:  proxyAddr,
	}

	var connection internal.ConnectionHandler

	// 3. 处理插件（如果指定）
	if *pluginConfig != "" {
		parts := strings.SplitN(*pluginConfig, ":", 2)
		if len(parts) != 2 {
			log.Fatal("Invalid plugin format. Use name:{\"key\":\"value\"}")
		}
		pluginName, configJSON := parts[0], []byte(parts[1])
		log.Printf("Attempting to load plugin: %s", pluginName)
		p := plugins.GetPlugin(pluginName)
		if p == nil {
			log.Fatalf("Plugin '%s' not available or not registered", pluginName)
		}

		// 调用插件的 Configure，传入基础配置
		// Configure 内部会创建并初始化包含正确地址的 CustomConnection
		var err error
		connection, err = p.Configure(configJSON, baseCfg) // 忽略返回的 config (如果不需要)
		connection.SetHooks(connection)                    // 将 connection 传递给 CustomConnection 作为 Hooks
		if err != nil {
			log.Fatalf("Configure plugin '%s' failed: %v", pluginName, err)
		}
		if connection == nil {
			log.Fatalf("Configure plugin '%s' returned nil connection", pluginName)
		}
		log.Printf("Using connection connection provided by plugin: %s", pluginName)

	} else {
		// 4. 如果没有插件，创建默认的 internal.Connection 作为 connection
		log.Println("No plugin specified, using default connection connection.")
		connection = &internal.Connection{ // 创建默认实现实例
			RemoteAddr: baseCfg.RemoteAddr,
			ProxyAddr:  baseCfg.ProxyAddr,
		}
	}

	if *baseCfg.RemoteAddr == "" {
		fmt.Println("Error: -remote flag is required")
		flag.Usage()
		os.Exit(1)
	}

	log.Printf("Starting RTMPS proxy...")
	log.Printf("Listening on: %s", *baseCfg.ListenAddr)
	log.Printf("Forwarding to: %s", *baseCfg.RemoteAddr)
	if *baseCfg.ProxyAddr != "" {
		log.Printf("Using Proxy: %s", *baseCfg.ProxyAddr)
	}

	listener, err := net.Listen("tcp", *baseCfg.ListenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", *baseCfg.ListenAddr, err)
	}
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)

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
