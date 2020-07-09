package main

import (
	"flag"
)

var (
	BuildVersion string
	BuildTime    string
	BuildName    string
	CommitID     string
	showVer      bool
)

func init() {
	flag.BoolVar(&showVer, "version", false, "show version")
}
