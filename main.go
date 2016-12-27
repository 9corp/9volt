package main

import (
	"strings"
	"sync"

	"github.com/InVisionApp/rye"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/9corp/9volt/alerter"
	"github.com/9corp/9volt/api"
	"github.com/9corp/9volt/cluster"
	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/dal"
	"github.com/9corp/9volt/director"
	"github.com/9corp/9volt/manager"
	"github.com/9corp/9volt/state"
	"github.com/9corp/9volt/util"
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

	// Create an initial dal client
	dalClient, err := dal.New(*etcdPrefix, *etcdMembers)
	if err != nil {
		log.Fatalf("Unable to start initial etcd client: %v", err.Error())
	}

	// Load our configuration
	cfg := config.New(*listenAddress, *etcdPrefix, *etcdMembers, dalClient)

	if err := cfg.Load(); err != nil {
		log.Fatalf("Unable to load configuration from etcd: %v", err.Error())
	}

	// Perform etcd layout validation
	if errorList := cfg.ValidateDirs(); len(errorList) != 0 {
		log.Fatalf("Unable to complete etcd layout validation: %v", strings.Join(errorList, "; "))
	}

	// Create necessary channels
	clusterStateChannel := make(chan bool)
	distributeChannel := make(chan bool)
	messageChannel := make(chan *alerter.Message)
	monitorStateChannel := make(chan *state.Message)

	// Start cluster engine
	cluster, err := cluster.New(cfg, clusterStateChannel, distributeChannel)
	if err != nil {
		log.Fatalf("Unable to instantiate cluster engine: %v", err.Error())
	}

	if err := cluster.Start(); err != nil {
		log.Fatalf("Unable to complete cluster engine initialization: %v", err.Error())
	}

	// start director (check distributor)
	director, err := director.New(cfg, clusterStateChannel, distributeChannel)
	if err != nil {
		log.Fatalf("Unable to instantiate director: %v", err.Error())
	}

	if err := director.Start(); err != nil {
		log.Fatalf("Unable to complete director initialization: %v", err.Error())
	}

	// start manager
	manager, err := manager.New(cfg, messageChannel, monitorStateChannel)
	if err != nil {
		log.Fatalf("Unable to instantiate manager: %v", err.Error())
	}

	if err := manager.Start(); err != nil {
		log.Fatalf("Unable to complete manager initialization: %v", err.Error())
	}

	// start the alerter
	alerter := alerter.New(cfg, messageChannel)

	if err := alerter.Start(); err != nil {
		log.Fatalf("Unable to complete alerter initialization: %v", err.Error())
	}

	// start the state dumper
	state := state.New(cfg, monitorStateChannel)

	if err := state.Start(); err != nil {
		log.Fatalf("Unable to complete state initialization: %v", err.Error())
	}

	// create a new middleware handler
	mwHandler := rye.NewMWHandler(rye.Config{})

	// start api server
	apiServer := api.New(cfg, mwHandler, version)
	go apiServer.Run()

	// Naming convention; intended module purpose
	//
	// D - DONE
	// S - SKIP
	// P - PENDING
	// ? - UNSURE
	//
	// [ P ] api       --  main API entry point
	// [ D ] director  --  performs check distribution
	// [ D ] manager   --  manages check lifetime
	// [ D ] cluster   --  performs leader election; heartbeat
	// [ ? ] fetcher   --  fetch statistics from outside sources
	//                     (needs additional discussion)
	// [ D ] alerter   --  send alerts to various destinations
	// [ P ] state     --  periodically dump state to etcd [ JESSE ]
	// [ D ] config    --  configuration validation and loading
	//

	log.Infof("9volt has started! API address: %v MemberID: %v", "http://"+
		*listenAddress, util.GetMemberID(*listenAddress))

	wg.Wait()
}
