package utils

import (
	"errors"
)

// Proxy error
var (
	InvalidProxy              = errors.New("invalid proxy address")
	InvalidProxyScheme        = errors.New("unsupported proxy scheme")
	InvalidProxyAuth          = errors.New("invalid proxy username or password")
	FailedToCreateProxyDialer = errors.New("failed to create proxy dialer")
)

// Init Connection error
var (
	UnSupportedScheme           = errors.New("unsupported scheme")
	ErrorToExtractParams        = errors.New("error to extract params")
	FailedToConnectRemoteServer = errors.New("failed to connect to remote server")
	FailedToEstablishTLS        = errors.New("failed to establish TLS")
)

var (
	FailedBeforeEstablishConnection = errors.New("failed before establish connection")
	FailedAfterCloseConnection      = errors.New("failed after connection callback")
)
