package internal

import (
	"errors"
)

// Proxy error
var (
	InvalidProxy              = errors.New("invalid proxy address")
	InvalidProxyScheme        = errors.New("unsupported proxy scheme")
	InvaildProxyPassword      = errors.New("invalid proxy password")
	FailedToCreateProxyDialer = errors.New("failed to create proxy dialer")
)

// Init Connection error
var (
	UnSupportedScheme           = errors.New("unsupported scheme")
	FailedToConnectRemoteServer = errors.New("failed to connect to remote server")
	FailedToEstablishTLS        = errors.New("failed to establish TLS")
)
