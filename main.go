package main

import (
	"flag"
	"fmt"
	env "github.com/Netflix/go-env"
	"k8s.io/klog"
	"net/http"
	"os"
)

var config = Config{
	UseTLS:                          true,
	CertFile:                        "/etc/skac/tls.crt",
	KeyFile:                         "/etc/skac/tls.key",
	TLSClientAuth:                   false,
	TriggerENV:                      false,
	SWImage:                         "ilemontech/skywalking-java-agent",
	SWAgentCollectorBackendServices: "skywalking-aop.skywalking:11800",
	SWJavaENVName:                   "JAVA_TOOL_OPTIONS",
}

func main() {
	if _, err := env.UnmarshalFromEnviron(&config); err != nil {
		klog.Error(err)
		return
	}
	config.addFlags()

	klog.InitFlags(nil)

	flag.Parse()

	showVersion()

	http.HandleFunc("/health", health)
	http.HandleFunc("/", serveMutatePods)

	fmt.Println("Starting")

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

func showVersion() {
	if showVer {
		fmt.Printf("build name:\t%s\n", BuildName)
		fmt.Printf("build ver:\t%s\n", BuildVersion)
		fmt.Printf("build time:\t%s\n", BuildTime)
		fmt.Printf("Commit ID:\t%s\n", CommitID)
		os.Exit(0)
	}
}
