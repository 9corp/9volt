# Dev
- Assuming dev work is happening on OS X
- Run `bootstrap.sh`
    + This will install: `homebrew`, `golang`, `etcd` and various go packages
- If you see a "All looks well!", start `9volt` by doing the following:
    + `godep go run main.go -e http://localhost:2379`
- You should be good to go!

## Random musings
- Use `gofmt`
- Use the built-in race detector (`go run -race`)
- `foo.Start()` for things that will complete (ie. may launch goroutines, etc.)
- `foo.Run()` for things that will *not* complete and should (probably) be launched in a goroutine
- Use bailout blocks/negative logic/negated if blocks when possible
- Make an effort to use interfaces - it will make testing *a lot* easier
- Interface types should be prefixed with an `I` (sorry gophers!)
- Using interfaces will allow you to use `counterfeiter` for generating fakes
    + https://github.com/maxbrunsfeld/counterfeiter
- At minimum, use an assertion library when writing tests (ie. https://github.com/stretchr/testify)
- Make an effort to ensure everything is unit testable first
- Tests should be tagged as follows (at the top of each test file):
    + For unit tests
        * `// +build unit`
    + For integration tests
        * `// +build integration`
    + For functional tests
        * `// +build functional`

## What does what?

### API [ DONE ]
API; primary way to interact with `9volt`.

### Cluster [ DONE ]
Performs leader election; heartbeat.

### Director [ DONE ]
Performs check (re)distribution between all cluster members.

### Manager [ PENDING ] @dselans
Manages check lifetime (ie. start/stop/reload).

### Monitor [ ? ]
Not sure if this is still needed since checks manager themselves. *Needs additional discussion.*

### Fetcher [ ? ]
Fetch statistics/metrics from outside sources and expose them to checks. *Needs additional discussion.*

### Alerter [ ? ]
Sends alerts to various destinations. Not sure what this will end up looking like. *Needs additional discussion.*

### State [ PENDING ] @jesse
Periodically dump state to etcd.

### Config [ DONE ]
Configuration loading and validation.

## Simplified flow/9volt cluster mechanism

1. Start up cluster, director, manager.
2. Cluster decides whether it is leader or not.
    * If it determines it's the leader:
        * tell director that it has become the leader
    * If it determines it's not the leader:
        * director idles/does nothing; manager watches its own config namespace
3. Director redistributes check configuration between all cluster members (including itself).
4. Individual managers notice changes inside their members config dirs and start/stop checkers as needed.

