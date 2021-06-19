package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	const rootPEM = `
-----BEGIN CERTIFICATE-----
MIIDcTCCAlmgAwIBAgIUY+388BP9ShDl7Nmn90ZTs+6O7FgwDQYJKoZIhvcNAQEL
BQAwLTELMAkGA1UEBhMCQVUxHjAcBgNVBAMMFWxvY2FsaG9zdC5sb2NhbGRvbWFp
bjAeFw0yMTA2MTkwNTQzNDdaFw0yMjA2MTkwNTQzNDdaMC0xCzAJBgNVBAYTAkFV
MR4wHAYDVQQDDBVsb2NhbGhvc3QubG9jYWxkb21haW4wggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDDdziLBhn1e/pqkkPD1WgbszXVTdhFg9FUcYdLhNIs
X0G4xfydu+R7wVp4aWMLvIKrT52Kc4zdd94pWMKqFeMZoAPp5vCjls+U/iYB1YNW
Er3e3SljejcIrdQWfRTbyiGB2wxQz0TPjh9lCpgC0SrcyhuXC9pbULxz2Owv+DGK
pmeB9eoeOQuIVxafEMw4a2iXnDAFImiZQkUOE9CtsR8oggCbBBcQ/pmauSC0iwN9
dUKrCtgMUIsovrEbqGoTjEbMDj8NtXku/hj7peR56rAswtVUU/4XmL38uzAQ8Xz6
sL/HnNhMjQrrdGx1BiMnX+ImfYpF1ASXCNocUiklYb5FAgMBAAGjgYgwgYUwHQYD
VR0OBBYEFDFgh7mBLD9V/1Dy9NTuuemJ0PbVMB8GA1UdIwQYMBaAFDFgh7mBLD9V
/1Dy9NTuuemJ0PbVMA8GA1UdEwEB/wQFMAMBAf8wIAYDVR0RBBkwF4IVbG9jYWxo
b3N0LmxvY2FsZG9tYWluMBAGA1UdIAQJMAcwBQYDKgMEMA0GCSqGSIb3DQEBCwUA
A4IBAQBG8iWo09EAlR/RtK7CbXUtj0DAv6W9kEFuCX6xjTCxJaj9a5SaskIT82Yj
soTG/i5PzUT91bQX3rR2M0uHwvsS/LSukYqlpdLqS+EjbsBr9JfNkJVlrvuVU753
MPSREICyKpnUbaybUsDksqLEpd7u3Q+rg2xWbANh7eACmRLoeEXQ9mLhCuLu6lFc
ojXeec5Nkl4uUisGcC4By60bAhL3FPnSgPLI3NA0OnRfpJOIFK6g5BrkJySowMyd
Vb+14Ce/IuGDCHtq04AWdBwEe/iVDr04iTekU2kIM9zAkftiobHKNKfeL8ZJ2w2P
jQeBS0VXS1m0StVdaEpV9aP2v8tU
-----END CERTIFICATE-----`
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(rootPEM))
	if !ok {
		panic("failed to parse root certificate")
	}

	tc := &tls.Config{
		Rand:                  nil,
		Time:                  nil,
		Certificates:          nil,
		NameToCertificate:     nil,
		GetCertificate:        nil,
		GetClientCertificate:  nil,
		GetConfigForClient:    nil,
		VerifyPeerCertificate: nil,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if cs.Version == tls.VersionTLS10 {
				fmt.Println("Negotiated TLS Version: 1.0")
			}
			if cs.Version == tls.VersionTLS11 {
				fmt.Println("Negotiated TLS Version: 1.1")
			}

			if cs.Version == tls.VersionTLS12 {
				fmt.Println("Negotiated TLS Version: 1.2")
			}

			if cs.Version == tls.VersionTLS13 {
				fmt.Println("Negotiated TLS Version: 1.3")
			}

			if cs.Version == tls.VersionSSL30 {
				fmt.Println("Negotiated SSL Version: 3.0")
			}
			return nil
		},
		RootCAs:                     roots,
		NextProtos:                  nil,
		ServerName:                  "localhost.localdomain",
		ClientAuth:                  0,
		ClientCAs:                   nil,
		InsecureSkipVerify:          false,
		CipherSuites:                nil,
		PreferServerCipherSuites:    true,
		SessionTicketsDisabled:      false,
		SessionTicketKey:            [32]byte{},
		ClientSessionCache:          nil,
		MinVersion:                  0,
		MaxVersion:                  0,
		CurvePreferences:            nil,
		DynamicRecordSizingDisabled: false,
		Renegotiation:               0,
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:                  nil,
			DialContext:            nil,
			Dial:                   nil,
			DialTLSContext:         nil,
			DialTLS:                nil,
			TLSClientConfig:        tc,
			TLSHandshakeTimeout:    0,
			DisableKeepAlives:      false,
			DisableCompression:     false,
			MaxIdleConns:           0,
			MaxIdleConnsPerHost:    0,
			MaxConnsPerHost:        0,
			IdleConnTimeout:        0,
			ResponseHeaderTimeout:  0,
			ExpectContinueTimeout:  0,
			TLSNextProto:           nil,
			ProxyConnectHeader:     nil,
			GetProxyConnectHeader:  nil,
			MaxResponseHeaderBytes: 0,
			WriteBufferSize:        0,
			ReadBufferSize:         0,
			ForceAttemptHTTP2:      false,
		},
	}
	resp, err := client.Get("https://localhost.localdomain:1443/") //server.URL)
	if err != nil {
		log.Fatalf("Failed to get URL: %v", err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to get body: %v", err)
	}
	fmt.Printf("body: %s\n", b)
}
