//go:build bilibili

package Bilibili

import (
	"github.com/CuteReimu/bilibili/v2"
	"rtmpproxy/internal"
)

type CustomInterceptor struct {
	BaseCfg  *internal.Config
	Cookie   string `json:"cookie"`
	RoomID   int    `json:"room_id"`
	AreaV2   int    `json:"area_v2"`
	Platform string `json:"platform,omitempty"` // 默认 android_link
	client   *Client
}

func (c *CustomInterceptor) ApplicationStart() error {
	var err error
	if c.client == nil {
		cli, err := CreateClient(c.Cookie)
		if err != nil {
			return err
		}
		c.client = cli
	}
	startLiveParam := bilibili.StartLiveParam{
		RoomId:   c.RoomID,
		AreaV2:   c.AreaV2,
		Platform: c.Platform,
	}
	startLiveResult, err := c.client.StartLive(startLiveParam)
	if err != nil {
		return StartLiveFailed
	}
	rtmpAddr := startLiveResult.Rtmp.Addr + startLiveResult.Rtmp.Code
	c.BaseCfg.RemoteAddr = &rtmpAddr
	return nil
}

func (c *CustomInterceptor) BeforeEstablishTCPConnection() error {
	return nil
}

func (c *CustomInterceptor) AfterRTMPHandshake() error {
	return nil
}

func (c *CustomInterceptor) AfterCloseTCPConnection() error {
	_, err := c.client.StopLive(bilibili.StopLiveParam{
		RoomId: c.RoomID,
	})
	if err != nil {
		return StopLiveFailed
	}
	return nil
}
