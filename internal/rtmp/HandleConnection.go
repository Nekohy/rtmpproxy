package rtmp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	amf "github.com/zhangpeihao/goamf"
)

type rtmpChunkHeader struct {
	format    uint32
	csid      uint32
	timestamp uint32
	length    uint32
	typeid    uint32
	streamid  uint32
}

// handleMessages 处理Client数据包，修改后的直接转发给Server
func (c *RTMPConnection) HandleMessages() error {
	var (
		maxChunkSize = 128
		usecopy      = false

		lastch  rtmpChunkHeader
		payload []byte
		nread   int
	)

	for !usecopy {
		ch, err := rtmpReadHeader(c.ClientConn)
		if err != nil {
			return err
		}
		if nread != 0 && ch.csid != lastch.csid {
			return fmt.Errorf("unsupport multi-chunkstream at a time")
		}

		switch ch.format {
		case 1:
			ch.streamid = lastch.streamid
		case 2:
			ch.length = lastch.length
			ch.typeid = lastch.typeid
			ch.streamid = lastch.streamid
		case 3:
			ch.timestamp = lastch.timestamp
			ch.length = lastch.length
			ch.typeid = lastch.typeid
			ch.streamid = lastch.streamid
		}
		lastch = *ch

		if len(payload) != int(ch.length) {
			payload = make([]byte, ch.length)
		}

		n := maxChunkSize
		if rem := len(payload) - nread; rem < maxChunkSize {
			n = rem
		}

		_, err = io.ReadFull(c.ClientConn, payload[nread:nread+n])
		if err != nil {
			return err
		}
		nread += n
		if nread < len(payload) {
			continue
		}

		switch ch.typeid {
		case 1:
			if len(payload) != 4 {
				return fmt.Errorf("invalid type 0 payload size: %d", len(payload))
			}
			maxChunkSize = int(binary.BigEndian.Uint32(payload))
			if maxChunkSize <= 0 {
				return fmt.Errorf("invalid chunk size: %d", maxChunkSize)
			}
		case 20:
			payload, usecopy, err = c.handleRtmpCommand(payload)
			if err != nil {
				return err
			}
		}
		err = writeRtmpMessage(c.ServerConn, ch, payload, maxChunkSize)
		if err != nil {
			return err
		}
		payload = nil
		nread = 0
	}
	return nil
}

// handleRtmpCommand 处理并修改rtmp命令
func (c *RTMPConnection) handleRtmpCommand(payload []byte) ([]byte, bool, error) {
	br := bytes.NewReader(payload)
	command, err := amf.ReadString(br)
	if err != nil {
		return nil, false, err
	}
	transid, err := amf.ReadDouble(br)
	if err != nil {
		return nil, false, err
	}
	args := make([]interface{}, 0, 1)
	for br.Len() > 0 {
		v, err := amf.ReadValue(br)
		if err != nil {
			return nil, false, err
		}
		args = append(args, v)
	}

	usecopy := false
	switch command {
	case "connect":
		obj := args[0].(amf.Object)
		obj["app"] = c.appName
		obj["swfUrl"] = c.playUrl
		obj["tcUrl"] = c.playUrl
	case "releaseStream", "FCPublish":
		args[1] = c.streamName
	case "publish":
		args[1] = c.streamName
		usecopy = true // 后续不再处理数据包，todo：以后要处理，用来Callback，其实也不太用，TCP数据流关闭做标志就好了
	}
	buf := bytes.NewBuffer(nil)
	_, _ = amf.WriteString(buf, command)
	_, _ = amf.WriteDouble(buf, transid)
	for _, arg := range args {
		_, _ = amf.WriteValue(buf, arg)
	}
	return buf.Bytes(), usecopy, nil
}
