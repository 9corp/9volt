package main

import (
	"sync"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/9corp/9volt/config"
	// "github.com/9corp/9volt/dal"
)

var (
	listenAddress = kingpin.Flag("listen", "Address for 9volt's API to listen on").Short('l').Default("0.0.0.0:8080").String()
	etcdPrefix    = kingpin.Flag("etcd-prefix", "Prefix that 9volt's configuration is stored under in etcd").Short('p').Default("9volt").String()
	etcdMembers   = kingpin.Flag("etcd-members", "List of etcd cluster members").Short('e').Required().Strings()
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

	// Load our configuration
	cfg := config.New(*listenAddress, *etcdPrefix, *etcdMembers)

	if err := cfg.Load(); err != nil {
		log.Fatalf("Unable to load configuration from etcd: %v", err.Error())
	}

	// Validate configuration in etcd
	// dalClient, err := dal.New(*etcdPrefix, *etcdMembers)
	// if err != nil {
	// 	log.Fatalf("Unable to instantiate dal client: %v", err.Error())
	// }

	// if err := dalClient.ValidatePaths(); err != nil {
	// 	log.Fatalf("Unable to validate all paths in etcd: %v", err.Error())
	// }

	// Start cluster engine
	// cluster := Cluster.New(cfg)

	// if err := cluster.Init(); err != nil {
	// 	log.Fatalf("Unable to complete cluster engine initialization: %v", err.Error())
	// }

	// Naming convention; intended module purpose

	// api       --  main API entry point
	// director  --  performs check distribution
	// manager   --  manages check lifetime
	// cluster   --  performs leader election; heartbeat
	// monitor   --  perform actual monitoring
	// fetcher   --  fetch statistics from outside sources
	// alerter   --  send alerts to various destinations
	// state     --  periodically dump state to etcd
	// config    --  configuration validation and loading

	wg.Wait()
}
