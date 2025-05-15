package main

import (
	"flag"
	"log"
	"net"
	"rtmpproxy/internal"
	"rtmpproxy/internal/plugins"
	"rtmpproxy/internal/rtmp"
	_ "rtmpproxy/plugins/Test"
	"rtmpproxy/utils"
	"strings"
)

func main() {
	listenAddr := flag.String("listen", ":1935", "Local address to listen on (e.g., :1935)")
	remoteAddr := flag.String("remote", "", "Remote RTMPS server")
	proxyAddr := flag.String("proxy", "", "Socks5 proxy server address (e.g., socks5://username:password@127.0.0.1:7890)")
	pluginConfig := flag.String("plugin", "", `plugin config e.g. "test:{\"message\":\"hello world\"}"`)
	insecureSkipVerify := flag.Bool("ignore", false, "skip TLS certificate verification")
	flag.Parse()

	if *pluginConfig == "" && *remoteAddr == "" {
		flag.Usage()
		log.Fatal("Error: Remote Addr or Plugin is required")
	}

	baseCfg := &internal.Config{
		ListenAddr:         listenAddr,
		RemoteAddr:         remoteAddr,
		ProxyAddr:          proxyAddr,
		InsecureSkipVerify: *insecureSkipVerify,
	}

	var interceptor plugins.Interceptor
	if *pluginConfig != "" {
		parts := strings.SplitN(*pluginConfig, ":", 2)
		if len(parts) != 2 {
			log.Fatal("Invalid plugin format. Use name:{\"key\":\"value\"}")
		}
		pluginName, configJSON := parts[0], []byte(parts[1])
		log.Printf("Attempting to load plugin: %s", pluginName)
		p := plugins.GetPlugin(pluginName)
		if p == nil {
			log.Fatalf("plugin '%s' not available or not registered", pluginName)
		}
		var err error
		interceptor, err = p.Configure(configJSON, baseCfg)
		if err != nil {
			log.Fatalf("Configure plugin '%s' failed: %v", pluginName, err)
		}
	} else {
		interceptor = &plugins.DefaultInterceptor{}
	}
	err := interceptor.ApplicationStart()
	if err != nil {
		log.Fatalf("Interceptor ApplicationStart failed: %v", err)
		return
	}
	if *baseCfg.RemoteAddr == "" {
		log.Fatalf("Error: Remote Addr is required")
	}

	log.Printf("Starting RTMPS proxy...")
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
		go func(ClientConn net.Conn) {
			// 连接远程RTMP服务器
			err := interceptor.BeforeEstablishTCPConnection()
			if err != nil {
				log.Fatalf("Interceptor BeforeEstablishTCPConnection failed: %v", err)
				return
			}
			ServerConn, err := baseCfg.ConnectRemoteAddress()
			if err != nil {
				log.Printf("Failed to connect remote RTMP server: %v", err)
				return
			}
			defer func(ServerConn net.Conn) {
				_ = ServerConn.Close()
			}(ServerConn)

			// 在 goroutine 内部调用 HandleClient
			appName, streamName, playUrl, err := utils.GetLinkParams(baseCfg.RemoteURL)
			rtmpConnection := rtmp.CreateRTMPInstance(ClientConn, ServerConn, appName, playUrl, streamName)

			err = rtmpConnection.RTMPHandshake()
			if err != nil {
				log.Printf("RTMP handshake failed: %v", err)
				return
			}
			err = interceptor.AfterRTMPHandshake()
			if err != nil {
				log.Fatalf("Interceptor AfterRTMPHandshake failed: %v", err)
				return
			}
			err = rtmpConnection.Serve()
			err = interceptor.AfterCloseTCPConnection()
			if err != nil {
				log.Fatalf("Interceptor AfterCloseTCPConnection failed: %v", err)
				return
			}
			// 将错误（可能是 nil）发送到通道
			errChan <- err
		}(clientConn) // 将 clientConn 作为参数传递给 goroutine 的闭包
	}
}
