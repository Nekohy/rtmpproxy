package rtmp

import (
	"encoding/binary"
	"io"
)

func (h *rtmpChunkHeader) asBytes() []byte {
	n := 1
	csid := h.csid
	if csid >= 256+64 {
		csid = 1
		n += 2
	} else if csid >= 64 {
		csid = 0
		n += 1
	}
	switch h.format {
	case 0:
		n += 11
	case 1:
		n += 7
	case 2:
		n += 3
	}
	if h.timestamp >= 0xffffff {
		n += 4
	}
	data := make([]byte, n)
	data[0] = byte((h.format << 6) | csid)
	p := 1
	if csid <= 1 {
		binary.BigEndian.PutUint16(data[1:], uint16(h.csid-64))
		p += int(csid) + 1
	}
	ts := h.timestamp
	exts := uint32(0)
	if ts >= 0xffffff {
		ts = 0xffffff
		exts = ts
	}
	switch h.format {
	case 0, 1:
		binary.BigEndian.PutUint32(data[p:], ts<<8)
		binary.BigEndian.PutUint32(data[p+3:], h.length<<8)
		data[p+6] = byte(h.typeid)
		if h.format == 0 {
			binary.LittleEndian.PutUint32(data[p+7:], h.streamid)
			p += 11
		} else {
			p += 7
		}
	case 2:
		binary.BigEndian.PutUint16(data[p:], uint16(ts>>8))
		data[p+2] = byte(h.timestamp & 0xff)
		p += 3
	}
	if ts == 0xffffff && h.format != 3 {
		binary.BigEndian.PutUint32(data[p:], exts)
	}
	return data
}

func rtmpReadHeader(r io.Reader) (*rtmpChunkHeader, error) {
	var buf [18]byte
	_, err := io.ReadFull(r, buf[:1])
	if err != nil {
		return nil, err
	}
	format := uint32(buf[0] >> 6)
	csid := uint32(buf[0] & 0x3f)
	n := 0
	if csid <= 1 {
		n += int(csid) + 1
	}
	switch format {
	case 0:
		n += 11
	case 1:
		n += 7
	case 2:
		n += 3
	}
	ch := &rtmpChunkHeader{
		format: format,
		csid:   csid,
	}
	if n > 0 {
		_, err := io.ReadFull(r, buf[:n])
		if err != nil {
			return nil, err
		}
		p := 0
		switch csid {
		case 0:
			ch.csid = uint32(buf[0]) + 64
			p = 1
		case 1:
			ch.csid = uint32(binary.BigEndian.Uint16(buf[:])) + 64
			p = 2
		}
		hbuf := buf[p:]
		switch format {
		case 0:
			_ = hbuf[:11]
			ch.timestamp = binary.BigEndian.Uint32(hbuf) >> 8
			ch.length = binary.BigEndian.Uint32(hbuf[3:]) >> 8
			ch.typeid = uint32(hbuf[6])
			ch.streamid = binary.LittleEndian.Uint32(hbuf[7:])
		case 1:
			_ = hbuf[:7]
			ch.timestamp = binary.BigEndian.Uint32(hbuf) >> 8
			ch.length = binary.BigEndian.Uint32(hbuf[3:]) >> 8
			ch.typeid = uint32(hbuf[6])
		case 2:
			_ = hbuf[:3]
			ch.timestamp = binary.BigEndian.Uint32(hbuf) >> 8
		}
		if ch.timestamp == 0xffffff {
			_, err := io.ReadFull(r, buf[:4])
			if err != nil {
				return nil, err
			}
			ch.timestamp = binary.BigEndian.Uint32(buf[:4])
		}
	}
	return ch, nil
}

func writeRtmpMessage(w io.Writer, ch *rtmpChunkHeader, payload []byte, chunkSize int) error {
	nwrote := 0
	for nwrote < len(payload) {
		if nwrote == 0 {
			ch.format = 0
			ch.length = uint32(len(payload))
		} else {
			ch.format = 3
		}
		_, err := w.Write(ch.asBytes())
		if err != nil {
			return err
		}
		n := chunkSize
		if len(payload)-nwrote < chunkSize {
			n = len(payload) - nwrote
		}
		_, err = w.Write(payload[nwrote : nwrote+n])
		if err != nil {
			return err
		}
		nwrote += n
	}
	return nil
}
