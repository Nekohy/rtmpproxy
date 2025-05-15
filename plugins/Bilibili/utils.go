package Bilibili

import (
	"bufio"
	"fmt"
	"github.com/CuteReimu/bilibili/v2"
	"os"
)

type Client struct {
	*bilibili.Client
}

// 创建客户端
func CreateClient(cookies string) (*Client, error) {
	client := &Client{
		Client: bilibili.New(),
	}
	if cookies != "" {
		client.SetCookiesString(cookies)
	} else {
		err := client.QRLogin()
		if err != nil {
			return nil, err
		}
	}
	return client, nil
}

// QRLogin 二维码登录, 并Print cookies
func (c *Client) QRLogin() (err error) {
	qrCode, err := c.GetQRCode()
	if err != nil {
		return LoginFailed
	}
	qrCode.Print()
	// 在控制台按回车键进行登录
	fmt.Println("please press enter key to continue after login with QR code,...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')

	result, err := c.LoginWithQRCode(bilibili.LoginWithQRCodeParam{
		QrcodeKey: qrCode.QrcodeKey,
	})
	if err != nil || result.Code != 0 {
		return LoginFailed
	}
	cookiesString := c.GetCookiesString()
	fmt.Println("Store your cookies for next use:", cookiesString)
	return nil
}
