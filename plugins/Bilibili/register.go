//go:build bilibili

package Bilibili

import (
	"encoding/json"
	"fmt"
	"rtmpproxy/internal"
	"rtmpproxy/internal/plugins"
)

type PluginConfig struct {
	CustomInterceptor CustomInterceptor
}

func init() {
	plugins.Register(&PluginConfig{})
}

func (p *PluginConfig) Name() string { return "bilibili" }

func (p *PluginConfig) Configure(config []byte, baseCfg *internal.Config) (plugins.Interceptor, error) {
	if err := json.Unmarshal(config, &p.CustomInterceptor); err != nil {
		return nil, err
	}
	if p.CustomInterceptor.RoomID == 0 {
		return nil, fmt.Errorf("room_id is required")
	}
	if p.CustomInterceptor.AreaV2 == 0 {
		return nil, fmt.Errorf("area_v2 is required,please check https://github.com/SocialSisterYi/bilibili-API-collect/blob/master/docs/live/live_area.md")
	}
	if p.CustomInterceptor.Platform == "" {
		p.CustomInterceptor.Platform = "android_link"
	}
	p.CustomInterceptor.BaseCfg = baseCfg
	customInterceptor := &p.CustomInterceptor
	return customInterceptor, nil
}
