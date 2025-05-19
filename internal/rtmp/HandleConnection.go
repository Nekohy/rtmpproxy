package rtmp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	amf "github.com/zhangpeihao/goamf"
	"io"
	"log"
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

	for !usecopy || c.forceHandle {
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
		if c.flashVer != "" {
			obj["flashVer"] = c.flashVer // "flashVer -> FMLE/3.0 (compatible; FMSc/1.0)" obs默认值
		}
		if c.rtmpType != "" {
			fmt.Printf("rtmpType: %s\n", c.rtmpType)
			obj["rtmpType"] = c.rtmpType
		}
		// log输出
		keys := []string{"app", "flashVer", "swfUrl", "tcUrl", "type"}
		var output string
		for _, k := range keys {
			output += fmt.Sprintf("%s=%s ", k, obj[k])
		}
		log.Println("RTMP Connect Params:", output)
	case "releaseStream", "FCPublish":
		args[1] = c.streamName
	case "publish":
		args[1] = c.streamName
		usecopy = true
	case "FCUnpublish":
		args[1] = c.streamName // 如果不forcehandle的话这里不会处理，不过也无所谓，TCP流也会关，只是减少特征
	}
	buf := bytes.NewBuffer(nil)
	_, _ = amf.WriteString(buf, command)
	_, _ = amf.WriteDouble(buf, transid)
	for _, arg := range args {
		_, _ = amf.WriteValue(buf, arg)
	}
	return buf.Bytes(), usecopy, nil
}
