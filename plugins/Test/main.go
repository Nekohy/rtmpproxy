package Test

import (
	"log"
)

type CustomInterceptor struct {
	message string
}

func (c *CustomInterceptor) ApplicationStart() error {
	log.Println("ApplicationStart Test:", c.message)
	return nil
}

func (c *CustomInterceptor) BeforeEstablishTCPConnection() error {
	log.Println("BeforeEstablishTCPConnection Test:", c.message)
	return nil
}

func (c *CustomInterceptor) AfterRTMPHandshake() error {
	log.Println("AfterRTMPHandshake Test:", c.message)
	return nil
}

func (c *CustomInterceptor) AfterCloseTCPConnection() error {
	log.Println("AfterCloseTCPConnection Test:", c.message)
	return nil
}
