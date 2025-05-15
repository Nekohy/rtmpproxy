package test

import (
	"encoding/json"
	"rtmpproxy/internal"
	"rtmpproxy/internal/plugins"
)

type PluginConfig struct {
	CustomInterceptor CustomInterceptor
}

func init() {
	plugins.Register(&PluginConfig{})
}

func (p *PluginConfig) Name() string { return "test" }

func (p *PluginConfig) Configure(config []byte, baseCfg *internal.Config) (plugins.Interceptor, error) {
	if err := json.Unmarshal(config, &p.CustomInterceptor); err != nil {
		return nil, err
	}
	customInterceptor := &p.CustomInterceptor
	return customInterceptor, nil
}
