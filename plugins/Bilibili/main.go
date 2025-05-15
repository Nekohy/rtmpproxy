package Bilibili

import "log"

type CustomInterceptor struct {
	message string
}

func (c *CustomInterceptor) ApplicationStart() error {
	log.Println("ApplicationStart test:", c.message)
	return nil
}

func (c *CustomInterceptor) BeforeEstablishTCPConnection() error {
	log.Println("BeforeEstablishTCPConnection test:", c.message)
	return nil
}

func (c *CustomInterceptor) AfterRTMPHandshake() error {
	log.Println("AfterRTMPHandshake test:", c.message)
	return nil
}

func (c *CustomInterceptor) AfterCloseTCPConnection() error {
	log.Println("AfterCloseTCPConnection test:", c.message)
	return nil
}
