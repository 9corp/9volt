package main

import (
	"sync"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	listenAddress = kingpin.Flag("listen", "Address for 9volt's API to listen on").Short('l').Default("0.0.0.0:8080").String()
	etcdPrefix    = kingpin.Flag("etcd-prefix", "Prefix that 9volt's configuration is stored under in etcd").Short('p').Default("9volt").String()
	etcdHosts     = kingpin.Flag("etcd-members", "List of etcd cluster members").Short('e').Strings()
	debug         = kingpin.Flag("debug", "Enable debug mode").Short('d').Bool()

	version string
)

func init() {
	log.SetLevel(log.InfoLevel)

	// Parse CLI stuff
	kingpin.Version(version)
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.VersionFlag.Short('v')
	kingpin.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	// Naming convention; intended module purpose

	// api      --  main API entry point
	// director --  performs check distribution
	// manager  --  manages check lifetime
	// cluster  --  performs leader election; heartbeat
	// monitor  --  perform actual monitoring
	// fetcher  --  fetch statistics from outside sources
	// alerter  --  send alerts to various destinations
	// state    --  periodically dump state to etcd

	wg.Wait()
}
