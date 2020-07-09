package main

import (
	"flag"
	"k8s.io/klog"
	"net/http"
	env "github.com/Netflix/go-env"
)

var config = Config{
	UseTLS:        false,
	CertFile:      nil,
	KeyFile:       nil,
	TLSClientAuth: false,
	triggerENV:    true,
}

func main() {

	if _, err := env.UnmarshalFromEnviron(config); err != nil {
		klog.Error(err)
		return
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
