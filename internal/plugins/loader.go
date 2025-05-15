package plugins

import (
	"rtmpproxy-re/internal"
)

type Plugin interface {
	Name() string                                                           // 插件名称
	Configure(config []byte, baseCfg *internal.Config) (Interceptor, error) // 返回一个 PluginHandler 实例
}

// 全局插件注册表
var plugins = make(map[string]Plugin)

func Register(p Plugin) {
	if _, exists := plugins[p.Name()]; exists {
		panic("plugin already registered: " + p.Name())
	}
	plugins[p.Name()] = p
}

func GetPlugin(name string) Plugin {
	return plugins[name]
}
