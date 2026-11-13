package main

import (
	"crypto/tls"
	"net/http"
)

func insecureTransport() *http.Transport {
	return &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec // explicit -i flag
}
