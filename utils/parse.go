package utils

import (
	"log"
	"net"
	"net/url"
	"strings"
)

const (
	defaultRTMPSPort = "443"
	defaultRTMPPort  = "1935"
	schemeRTMPS      = "rtmps"
	schemeRTMP       = "rtmp"
)

// ParseLink 端口添加并确定连接参数
func ParseLink(link string) (*url.URL, bool, error) {
	var useTLS bool
	convertUrl, err := url.Parse(link)
	if err != nil {
		return nil, false, err
	}
	switch strings.ToLower(convertUrl.Scheme) {
	case schemeRTMPS:
		useTLS = true
		if convertUrl.Port() == "" {
			convertUrl.Host = net.JoinHostPort(convertUrl.Hostname(), defaultRTMPSPort)
		}
	case schemeRTMP:
		useTLS = false
		if convertUrl.Port() == "" {
			convertUrl.Host = net.JoinHostPort(convertUrl.Hostname(), defaultRTMPPort)
		}
	default:
		return nil, false, UnSupportedScheme
	}

	return convertUrl, useTLS, nil
}

func GetLinkParams(u *url.URL) (string, string, string, error) {
	// u.Path 比如 "/appName/streamName"
	// u.RawQuery 用于Bilibili的直播地址
	var bilibiliPatch string
	if u.RawQuery != "" {
		bilibiliPatch = "?" + u.RawQuery
	}
	parts := strings.Split(strings.Trim(u.Path+bilibiliPatch, "/"), "/")
	if len(parts) < 2 {
		log.Fatalf("expected at least two path segments, got %d", len(parts))
		return "", "", "", ErrorToExtractParams
	}

	appName := parts[0]
	streamName := parts[1]
	playUrl := u.Host
	return appName, streamName, playUrl, nil
}
