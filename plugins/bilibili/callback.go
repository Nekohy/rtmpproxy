//go:build bilibili

package bilibili

import (
	"fmt"
	"rtmpproxy/internal"
)

type CustomConnection struct {
	Config Config
	internal.Connection
}

func (c *CustomConnection) BeforeEstablishConnection() error {
	fmt.Printf("test")
	return nil
}

func (c *CustomConnection) AfterCloseConnection() error {
	fmt.Printf("test")
	return nil
}
