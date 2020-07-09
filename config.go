package main

import (
	"crypto/tls"
	"flag"
	"k8s.io/klog"
)

// 配置信息
type Config struct {
	UseTLS        bool   `env:"SWKAC_USE_TLS"`
	CertFile      string `env:"SWKAC_TLS_CERT"`
	KeyFile       string `env:"SWKAC_TLS_KEY"`
	TLSClientAuth bool   `env:"SWKAC_TLS_CLIENT_AUTH"`
	triggerENV    bool   `env:"TRIGGER_ENV"`
}

func (c *Config) addFlags() {
	flag.BoolVar(&c.UseTLS, "use-tls", c.UseTLS,
		"run whit https.")
	flag.StringVar(&c.CertFile, "tls-cert-file", c.CertFile,
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated after server cert).")
	flag.StringVar(&c.KeyFile, "tls-private-key-file", c.KeyFile,
		"File containing the default x509 private key matching --tls-cert-file.")
	flag.BoolVar(&c.TLSClientAuth, "require-tls-client-auth", c.TLSClientAuth,
		"Require client auth with TLS, uses mutual tls on apiserver.")
}

func configTLS(config Config) *tls.Config {
	sCert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		klog.Fatal(err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}

	if config.TLSClientAuth {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig
}
