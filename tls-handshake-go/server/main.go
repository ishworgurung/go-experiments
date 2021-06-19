package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	DeprecatedTlsVersionWait = 15 * time.Second
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Write([]byte("This is an example server.\n"))
	})
	cfg := &tls.Config{
		MinVersion:               tls.VersionSSL30,
		MaxVersion:               tls.VersionTLS13,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		VerifyConnection: func(cs tls.ConnectionState) error {
			if cs.Version == tls.VersionSSL30 {
				fmt.Println("Negotiated SSL Version: 3.0")
				time.Sleep(DeprecatedTlsVersionWait)
				// StatsD UDP
			}

			if cs.Version == tls.VersionTLS10 {
				fmt.Println("Negotiated TLS Version: 1.0")
				time.Sleep(DeprecatedTlsVersionWait)
				// StatsD UDP
			}

			if cs.Version == tls.VersionTLS11 {
				fmt.Println("Negotiated TLS Version: 1.1")
				time.Sleep(DeprecatedTlsVersionWait)
				// StatsD UDP
			}

			if cs.Version == tls.VersionTLS12 {
				fmt.Println("Negotiated TLS Version: 1.2")
				time.Sleep(DeprecatedTlsVersionWait)
				// StatsD UDP
			}

			if cs.Version == tls.VersionTLS13 {
				fmt.Println("Negotiated TLS Version: 1.3")
				//time.Sleep(DeprecatedTlsVersionWait)
				// StatsD UDP
			}
			return nil
		},
		Rand: nil,
		Time: nil,
		//Certificates:                []tls.Certificate{cer},
		//NameToCertificate:     nil,
		GetCertificate:        nil,
		GetClientCertificate:  nil,
		GetConfigForClient:    nil,
		VerifyPeerCertificate: nil,
		//RootCAs:                     roots,
		NextProtos:             nil,
		ServerName:             "localhost.localdomain",
		ClientAuth:             0,
		ClientCAs:              nil,
		InsecureSkipVerify:     false,
		SessionTicketsDisabled: false,
		//SessionTicketKey:            [32]byte{},
		ClientSessionCache:          nil,
		DynamicRecordSizingDisabled: false,
		Renegotiation:               0,
		//KeyLogWriter:                w,
		//Rand:               zeroSource{}, // for reproducible output; don't do this.
		//InsecureSkipVerify: true,         // test server certificate is not trusted.
	}
	srv := &http.Server{
		Addr:         ":1443",
		Handler:      mux,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
	log.Fatal(srv.ListenAndServeTLS("server.crt", "server.key"))
}
