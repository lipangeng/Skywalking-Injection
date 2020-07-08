package main

import (
	"flag"
	"net/http"
)

func main() {
	config := Config{
		UseTLS: false,
	}

	config.addFlags()
	flag.Parse()

	http.HandleFunc("/", serveMutatePods)

	if config.UseTLS {
		server := &http.Server{
			Addr:      ":443",
			TLSConfig: configTLS(config),
		}
		_ = server.ListenAndServeTLS(config.CertFile, config.KeyFile)
	} else {
		server := &http.Server{
			Addr: ":80",
		}
		_ = server.ListenAndServe()
	}
}
