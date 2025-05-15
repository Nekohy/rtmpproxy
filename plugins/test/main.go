package test

import (
	"log"
)

type CustomInterceptor struct {
	Message string `json:"message"`
}

func (c *CustomInterceptor) ApplicationStart() error {
	log.Println("ApplicationStart test:", c.Message)
	return nil
}

func (c *CustomInterceptor) BeforeEstablishTCPConnection() error {
	log.Println("BeforeEstablishTCPConnection test:", c.Message)
	return nil
}

func (c *CustomInterceptor) AfterRTMPHandshake() error {
	log.Println("AfterRTMPHandshake test:", c.Message)
	return nil
}

func (c *CustomInterceptor) AfterCloseTCPConnection() error {
	log.Println("AfterCloseTCPConnection test:", c.Message)
	return nil
}
