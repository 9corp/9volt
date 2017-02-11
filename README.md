# 9volt

[![Build Status](https://travis-ci.org/9corp/9volt.svg?branch=master)](https://travis-ci.org/9corp/9volt)
[![Go Report Card](https://goreportcard.com/badge/github.com/9corp/9volt)](https://goreportcard.com/report/github.com/9corp/9volt)

A modern, distributed monitoring system written in Go.

### Another monitoring system? Why?
While there are a bunch of solutions for monitoring and alerting using time series data, there aren't many (or any?) modern solutions for 'regular'/'old-skool' remote monitoring similar to Nagios and Icinga.

`9volt` offers the following things out of the box:

- Single binary deploy
- Fully distributed
- Incredibly easy to scale to hundreds of thousands of checks
- Uses `etcd` for all configuration
- Real-time configuration pick-up (update etcd - `9volt` immediately picks up the change)
- Interval based monitoring (ie. run check XYZ every 1s, 1y, 1d or even 1ms)
- Natively supported monitors:
    - TCP
    - HTTP
    - Exec
- Natively supported alerters:
    - Slack
    - Pagerduty
    - Email
- RESTful API for querying current monitoring state and loaded configuration
- Comes bundled with a web app for a quick visual view of the cluster:
    + `./9volt-web -s 9volt-server-1.example.com, 9volt-server-2.example.com`
- Comes bundled with a binary tool to parse YAML based configs and push/sync them to etcd

### Usage
- Install/setup `etcd`
- Download latest `9volt` release
- For first time setup, run `./scripts/setup.sh`
- Start server: `./9volt -e http://etcd-server-1.example.com:2379 http://etcd-server-2.example.com:2379 http://etcd-server-3.example.com:2379`
- Optional: add `9volt` to be managed by `supervisord`, `upstart` or some other process manager

### H/A and scaling
Scaling `9volt` is incredibly simple. Launch another `9volt` service on a separate host and point it to the same `etcd` hosts as the main `9volt` service.

Your main `9volt` node will produce output similar to this when it detects a node join:

![node join](/assets/node-join.png?raw=true)

Checks will be automatically divided between the all `9volt` instances.

If one of the nodes were to go down, a new leader will be elected (*if the node that went down was the previous leader*) and checks will be redistributed among the remaining nodes.

This will produce output similar to this (and will be also available in the event stream via the API and UI):

![node-leave](/assets/node-leave.png?raw=true)

### API
API documentation can be found [here](docs/api/README.md).

### Minimum requirements (can handle ~1,000-3,000 <10s interval checks)
- 1 x 9volt instance (1 core, 256MB RAM)
- 1 x etcd node (1 core, 512MB RAM)

### Recommended (production) requirements (can handle 10,000+ <10s interval checks)
- 3 x 9volt instances (2+ cores, 512MB RAM)
- 3 x etcd nodes (2+ cores, 1GB RAM)

### Docs
Read through the [docs dir](docs/).

### Suggestions/ideas
Got a suggestion/idea? Something that is preventing you from using `9volt` over another monitoring system because of a missing feature? Submit an issue and we'll see what we can do!
