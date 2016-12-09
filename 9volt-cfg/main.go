package main

import (
	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/9corp/9volt/9volt-cfg/config"
	// "github.com/9corp/9volt/9volt-cfg/dal"
)

var (
	dirArg      = kingpin.Arg("dir", "Directory to search for 9volt YAML files").Required().String()
	prefixFlag  = kingpin.Flag("prefix", "Prefix that 9volt's configuration is stored under in etcd").Short('p').Default("9volt").String()
	hostsFlag   = kingpin.Flag("etcd-hosts", "List of etcd hosts").Short('e').Required().Strings()
	replaceFlag = kingpin.Flag("replace", "Do NOT verify if parsed config already exists in etcd (ie. replace everything)").Short('r').Bool()
	debugFlag   = kingpin.Flag("debug", "Enable debug mode").Short('d').Bool()

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
	// etcdClient, err := dal.New(*hostsFlag, *replaceFlag)
	// if err != nil {
	// 	log.Fatalf("Unable to create initial etcd client: %v", err.Error())
	// }

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

	for k, v := range configs.AlerterConfigs {
		log.Infof("Found %v alerter config", k)
		log.Infof("Contents: %v", string(v))
	}

	for k, v := range configs.MonitorConfigs {
		log.Infof("Found %v monitor config", k)
		log.Infof("Contents: %v", string(v))
	}

	// log.Infof("Pushing 9volt configs to etcd hosts: %v", *hostsFlag)

	// // push to etcd
	// info, err := etcdClient.Push(configs)
	// if err != nil {
	// 	log.Fatalf("Unable to push configs to etcd: %v", err.Error())
	// }

	// log.Infof(":party: Successfully pushed: %v monitor config(s); %v alerter config(s)", info.Monitor, info.Alerter)

	// if *replaceFlag {
	// 	log.Infof("Skipped replacing: %v monitor config(s); %v alerter config(s)", info.SkippedMonitor, info.SkippedAlerter)
	// }
}
