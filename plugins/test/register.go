package test

import (
	"encoding/json"
	"rtmpproxy/internal/plugins"
)

type PluginConfig struct {
	configJson configJson
}

type configJson struct {
	Message string `json:"message"`
}

func init() {
	plugins.Register(&PluginConfig{})
}

func (p *PluginConfig) Name() string { return "test" }

func (p *PluginConfig) Configure(config []byte) (plugins.Interceptor, error) {
	if err := json.Unmarshal(config, &p.configJson); err != nil {
		return nil, err
	}
	customInterceptor := &CustomInterceptor{
		message: p.configJson.Message,
	}
	return customInterceptor, nil
}
