# Overwatch

`9volt` is **heavily** dependent on *etcd*. A lot of it's basic functionality depends on it and if it happens to go away, 9volt can go into a pretty funky state. While `etcd` is fairly resilient, networks and other external factors may not be and a failure can occur. For such situations (which hopefully do not occur terribly often), we have `overwatch`.

`overwatch` is `9volt`'s own internal monitoring and recovery system. It is responsible for:

* Listening for failure alerts from internal components (such as the `alerter`, `cluster`, `manager`, etc.)
* Monitoring and waiting until the failed dependency recovers
* Stopping/starting components based on dependency condition

## What/how it do
All of the (major) internal components within `9volt` are instrumented to notify the `overwatch` component whenever they encounter an etcd failure.

When `overwatch` receives the failure message, it will do the following:

1. Request all main components to shutdown (including checkers).
2. This will cause the affected node to be removed from the cluster (causing a check re-distribution within the remaining nodes).
3. Toggle the healthstate on `/status/check` to return a non-200 response
4. `overwatch` will enter into a monitoring phase -- it will monitor etcd by establishing a watch and ensuring it does not encounter any errors for 30s (as of 2017.04.17).
5. If no errors were encountered with etcd within the 30s window, `overwatch` will start all of the main components once more.
6. The startup of the components will mimic that of a "fresh" `9volt` start -- 9volt will rejoin the cluster, forcing a new re-distribution to take place.

## Other misc bits
`overwatch` is enabled by default -- you don't have to do anything to make use of it. However, at the moment, it's not very configurable - that may change in the future, especially if we start seeing false-positives or seeing overly-eager/aggressive component shutdowns.
