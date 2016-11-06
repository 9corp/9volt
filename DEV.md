# Dev
- Assuming dev work is happening on OS X
- Use go version 1.6+
- Run `bootstrap.sh`
    + This will install: `homebrew`, `golang`, `etcd` and various go packages
- If you see a "All looks well!", start `9volt` by doing the following:
    + `go run main.go -e http://localhost:2379`
- You should be good to go!

## Random musings
- Use `gofmt`
- Use the built-in race detector (`go run -race`)
- `foo.Start()` for things that will not block and return control to the caller
- `foo.Run()` for things that will block forever and not return control to the caller
- Use bailout blocks/negative logic/negated if blocks when possible
- Make an effort to use interfaces - it will make testing *a lot* easier
- Interface types should be prefixed with an `I` (sorry gophers!)
- Using interfaces will allow you to use `counterfeiter` for generating fakes
    + https://github.com/maxbrunsfeld/counterfeiter
- Use `ginkgo` and `gomega` for testing
- Make an effort to ensure everything is unit testable first
- Tests should be tagged and named as follows (at the top of each test file):
    + For unit tests
        * No tag
        * `filename_test.go`
    + For integration tests
        * `// +build integration`
        * `filename_integration_test.go`
    + For functional tests
        * `// +build functional`
        * `filename_functional_test.go`

## What does what?

### API [ DONE ]
API; primary way to interact with `9volt`.

### Cluster [ DONE ]
Performs leader election; heartbeat.

### Director [ DONE ]
Performs check (re)distribution between all cluster members.

### Manager [ PENDING ] @dselans
Manages check lifetime (ie. start/stop/reload).

### Fetcher [ ? ]
Fetch statistics/metrics from outside sources and expose them to checks. *Needs additional discussion.*

### Alerter [ PENDING ]
Sends alerts to various destinations.

To simplify alerting story:

* Check configs reference <a href=""></a>n alert key (ie. "my-pagerduty-alert")
* Checks run into a changed state -> construct an alert message with given alert key ("my-pagerduty-alert")
* Alerter picks up the alert message from the channel and spins up the outbound alert in a separate goroutine
    - That spun-up alerter fetches the config for "my-pagerduty-alert" on the fly and uses it to send the alert
    - This can be optimized later to cache the alert-configuration (or maybe just do it from the start - depends on how much work is involved; if we already have a watcher, the entirety of alert configs could be stored in mem (?). Not sold on either way.)

### State [ PENDING ] @jesse
Periodically dump state to etcd.

### Config [ DONE ]
Configuration loading and validation.

## Simplified flow/9volt cluster mechanism

1. Start up cluster, director and manager goroutines.
2. Cluster decides whether it is leader or not.
    * If it determines it's the leader:
        * tell director goroutine that it has become the leader
    * If it determines it's not the leader:
        * director goroutine idles/does nothing; manager goroutine watches its own config namespace
3. Director goroutine redistributes check configuration between all cluster members (including itself).
4. Each 9volt's manager goroutines notice changes inside their members config dirs and start/stop checkers as needed.

# Potential AlerterConfig structure
```javascript
{
    "type"        : "pagerduty",
    "description" : "description about this alerter entry",
    "options"     : {
        "apikey"     : "1234567890",
        "custom-key" : "custom data used by the pagerduty alerter"
    }
}

