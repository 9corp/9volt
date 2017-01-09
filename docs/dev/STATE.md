State Info
==========

## Introduction to state

*state == individual monitor state information*

* Every check produces state information (on every `interval`)
* `9volt` dumps all of its monitor state info to etcd every `StateDumpInterval` (default `10s`)
* A state JSON blob is available for every running check
    - KEY: `/$PREFIX/state/$CHECK_NAME`
    - VALUE: JSON blob containing state info
* For convenience, state entries also contain the checks config

**Example State**

```shell
fullstop:9volt dselans$ curl -s http://localhost:2379/v2/keys/9volt/state/exec-check |json_pp
{
   "action" : "get",
   "node" : {
      "modifiedIndex" : 73954,
      "key" : "/9volt/state/exec-check",
      "value" : "{\"check\":\"exec-check\",\"owner\":\"365c6d2c\",\"status\":\"warning\",\"count\":6,\"message\":\"Command 'echo hello world' exited with a '0' return code, expected '1'\",\"date\":\"2016-12-26T19:48:04.204551513-08:00\",\"config\":{\"type\":\"exec\",\"description\":\"exec check test\",\"interval\":\"10s\",\"timeout\":\"5s\",\"expect\":\"world\",\"command\":\"echo\",\"args\":[\"hello\",\"world\"],\"return-code\":1,\"warning-threshold\":1,\"critical-threshold\":3,\"warning-alerter\":[\"primary-slack\"],\"critical-alerter\":[\"primary-email\"]}}",
      "createdIndex" : 73954
   }
}
```
