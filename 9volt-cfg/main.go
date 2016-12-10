package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/9corp/9volt/9volt-cfg/config"
	"github.com/9corp/9volt/9volt-cfg/dal"
)

var (
	dirArg      = kingpin.Arg("dir", "Directory to search for 9volt YAML files").Required().String()
	prefixFlag  = kingpin.Flag("prefix", "Prefix that 9volt's configuration is stored under in etcd").Short('p').Default("9volt").String()
	hostsFlag   = kingpin.Flag("etcd-hosts", "List of etcd hosts").Short('e').Required().Strings()
	replaceFlag = kingpin.Flag("replace", "Do NOT verify if parsed config already exists in etcd (ie. replace everything)").Short('r').Bool()
	dryrunFlag  = kingpin.Flag("dryrun", "Do NOT push any changes, just show me what you'd do").Short('d').Bool()
	debugFlag   = kingpin.Flag("debug", "Enable debug mode").Bool()

	version string
)

func init() {
	log.SetLevel(log.InfoLevel)

	// Parse CLI stuff
	kingpin.Version(version)
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.CommandLine.VersionFlag.Short('v')
	kingpin.Parse()

	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	etcdClient, err := dal.New(*hostsFlag, *prefixFlag, *replaceFlag, *dryrunFlag)
	if err != nil {
		log.Fatalf("Unable to create initial etcd client: %v", err.Error())
	}

	// verify if given dirArg is actually a dir
	cfg, err := config.New(*dirArg)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Infof("Fetching all 9volt configuration files in '%v'", *dirArg)

	yamlFiles, err := cfg.Fetch()
	if err != nil {
		log.Fatalf("Unable to fetch config files from dir '%v': %v", *dirArg, err.Error())
	}

	log.Info("Parsing 9volt config files")

	configs, err := cfg.Parse(yamlFiles)
	if err != nil {
		log.Fatalf("Unable to complete config file parsing: %v", err.Error())
	}

	log.Infof("Found %v alerter configs and %v monitor configs", len(configs.AlerterConfigs), len(configs.MonitorConfigs))
	log.Infof("Pushing 9volt configs to etcd hosts: %v", *hostsFlag)

	// push to etcd
	stats, errorList := etcdClient.Push(configs)
	if len(errorList) != 0 {
		log.Errorf("Encountered %v errors: %v", len(errorList), errorList)
	}

	pushedMessage := fmt.Sprintf("pushed %v monitor config(s) and %v alerter config(s)", stats.MonitorAdded, stats.AlerterAdded)
	skippedMessage := fmt.Sprintf("skipped replacing %v monitor config(s) and %v alerter config(s)", stats.MonitorSkipped, stats.AlerterSkipped)

	if *dryrunFlag {
		pushedMessage = "DRYRUN: Would have " + pushedMessage
		skippedMessage = "DRYRUN: Would have " + skippedMessage
	} else {
		pushedMessage = ":party: Successfully " + pushedMessage
		skippedMessage = "Successfully " + skippedMessage
	}

	log.Info(pushedMessage)

	if !*replaceFlag {
		log.Info(skippedMessage)
	}
}
