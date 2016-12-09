package dal

import (
// log "github.com/Sirupsen/logrus"
)

type DAL struct {
	Hosts   []string
	Replace bool
}

type PushStats struct {
	Monitor        int
	Alerter        int
	SkippedMonitor int
	SkippedAlerter int
}

func New(hosts []string, replace bool) (*DAL, error) {
	// Check if the given etcd servers are legit

	return &DAL{
		Hosts:   hosts,
		Replace: replace,
	}, nil
}

func (d *DAL) Push(configs []string) (*PushStats, error) {
	return &PushStats{}, nil
}
