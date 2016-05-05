# 9volt
A modern, distributed monitoring system written in Go.

### Another monitoring system? Why?
While there are a bunch of solutions for monitoring and alerting using time series data, there aren't many (or any?) modern solutions for 'regular'/'old-skool' remote monitoring similar to Nagios and Icinga.

`9volt` offers the following things out of the box:

- Single binary deploy
- Full H/A story (active/passive/passive/..)
- Use `etcd` for all configuration storage
- Real-time configuration pick-up (update etcd - `9volt` immediately picks up the change)
- RESTful API for querying current monitoring state and loaded configuration
- Comes bundled with a web app for a quick visual view of the cluster:
    + `./9volt-web -s 9volt-server-1.example.com, 9volt-server-2.example.com`

### Usage
- Setup an etcd cluster
- Download latest `9volt` release
- For first time setup, run `./9volt-init` and follow prompts for setup
- Start server: `./9volt -e http://etcd-server-1.example.com:2379 http://etcd-server-2.example.com:2379 http://etcd-server-3.example.com:2379`
- Optional: add `9volt` to be managed by `supervisord`, `upstart` or some other process manager

### Recommended requirements
- 3 x 9volt instances (4+ cores, 8GB RAM each)
- 1 x 5-node etcd cluster (2+ cores, 4GB RAM each)

### Minimum requirements
- 1 x 9volt instance (2+ cores, 4GB RAM each)
- 1 x 3-node etcd cluster (2+ cores, 2GB RAM each)

### Dev
- Assuming dev work is happening on OS X
- Run `bootstrap.sh`
    + This will install: `homebrew`, `golang`, `etcd` and various go packages
- If you see a "All looks well!", start `9volt` by doing the following:
    + `godep go run main.go -e http://localhost:2379`
- You should be good to go!
