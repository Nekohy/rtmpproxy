//go:build bilibili

package Bilibili

import (
	"github.com/CuteReimu/bilibili/v2"
	"log"
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
	if c.client == nil {
		cli, err := CreateClient(c.Cookie)
		if err != nil {
			return err
		}
		c.client = cli
	}
	log.Println("Success to load Bilibili Client")
	return nil
}

func (c *CustomInterceptor) BeforeEstablishTCPConnection() error {
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
	log.Printf("Success to start live, RoomID: %d, AreaV2: %d, Platform: %s", c.RoomID, c.AreaV2, c.Platform)
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
	log.Printf("Success to stop bilibili live, RoomID: %d", c.RoomID)
	return nil
}
