package plugins

type Interceptor interface {
	ApplicationStart() error             // 应用启动时的动作
	BeforeEstablishTCPConnection() error // TCP连接前的动作
	AfterRTMPHandshake() error           // RTMP握手后的动作
	AfterCloseTCPConnection() error      // TCP连接断开后的动作
}

type DefaultInterceptor struct{}

func (i *DefaultInterceptor) ApplicationStart() error {
	return nil
}

func (i *DefaultInterceptor) BeforeEstablishTCPConnection() error {
	return nil
}

func (i *DefaultInterceptor) AfterRTMPHandshake() error {
	return nil
}

func (i *DefaultInterceptor) AfterCloseTCPConnection() error {
	return nil
}
