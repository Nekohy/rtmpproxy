//go:build bilibili

package bilibili

import (
	"encoding/json"
	"rtmpproxy/internal"
	"rtmpproxy/plugins"
)

// Config Bilibili所需的参数
type Config struct {
	Cookies   string `json:"cookies"`
	Partition string `json:"partition"`
}

// PluginConfig 插件配置
type PluginConfig struct {
	RuntimeConfig internal.Config
	InputConfig   Config
	Callback      *CustomConnection
}

func init() {
	plugins.Register(&PluginConfig{})
}

func (p *PluginConfig) Name() string { return "bilibili" }

func (p *PluginConfig) Configure(config []byte, baseCfg *internal.Config) (internal.ConnectionHandler, error) {
	if err := json.Unmarshal(config, &p.InputConfig); err != nil {
		return nil, err
	}
	// todo 可以在这个位置修改baseCfg
	cfg := baseCfg
	// 回调实例
	p.Callback = &CustomConnection{
		Connection: internal.Connection{
			RemoteAddr: cfg.RemoteAddr,
			ProxyAddr:  cfg.ProxyAddr,
		},
		Config: p.InputConfig,
	}
	return p.Callback, nil
}
