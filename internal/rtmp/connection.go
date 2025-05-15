package rtmp

import (
	"fmt"
	"io"
	"log"
	"net"
)

type RTMPConnection struct {
	ClientConn net.Conn
	ServerConn net.Conn
	appName    string
	playUrl    string
	streamName string
}

type copyErr struct {
	dir string
	err error
}

func (c *RTMPConnection) RTMPHandshake() error {
	errs := make(chan copyErr, 2)

	copyfn := func(dir string, dst, src net.Conn) {
		_, err := io.CopyN(dst, src, 1+1536+1536)
		errs <- copyErr{dir, err}
	}
	log.Printf("Starting RTMP handshake...")

	go copyfn("C→S", c.ServerConn, c.ClientConn)
	go copyfn("S→C", c.ClientConn, c.ServerConn)

	cf1 := <-errs
	if cf1.err != nil {
		return fmt.Errorf("%s handshake error: %w", cf1.dir, cf1.err)
	}
	cf2 := <-errs
	if cf2.err != nil {
		return fmt.Errorf("%s handshake error: %w", cf2.dir, cf2.err)
	}
	log.Printf("RTMP handshake finished.")
	return nil
}

func (c *RTMPConnection) Serve() error {
	defer func(ServerConn net.Conn) {
		_ = ServerConn.Close()
	}(c.ServerConn)

	go func() {
		_, _ = io.Copy(c.ClientConn, c.ServerConn)
		c.ClientConn.Close()
		c.ServerConn.Close()
	}()

	err := c.HandleMessages()
	if err != nil {
		return err
	}
	// 剩余的字节直接转发
	_, err = io.Copy(c.ServerConn, c.ClientConn)
	c.ServerConn.Close()
	c.ClientConn.Close()
	return err
}

func CreateRTMPInstance(ClientConn net.Conn, ServerConn net.Conn, appName string, playUrl string, streamName string) *RTMPConnection {
	return &RTMPConnection{
		ClientConn: ClientConn,
		ServerConn: ServerConn,
		appName:    appName,
		playUrl:    playUrl,
		streamName: streamName,
	}
}
