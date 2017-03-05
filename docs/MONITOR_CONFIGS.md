# Monitor Config Documentation

"Special" types:

**duration**: a string that contains a decimal number and a unit suffix for 'ns', 'us', 'ms', 's', 'm' and 'h'. Read more [here](https://golang.org/pkg/time/#ParseDuration).

## Table of Contents 
- [Base Monitor Settings](#base-monitor-settings)
- [Member Tag](#member-tag-details)
- [Monitor Types](#monitor-types)
    - [Exec](#exec)
    - [HTTP](#http)
    - [TCP](#tcp)

## Base Monitor Settings 
There are a number of monitor configuration attributes that work for *all* monitor configs.

| Attribute          | Type         | Description |
|--------------------|--------------|---------------------------------------|
| type               | string       | check type                            |
| description        | string       | check description                     |
| host               | string       | target address                        |
| interval           | duration     | how often to perform check            |  
| timeout            | duration     | when to timeout the check             |
| port               | int          | target port                           |
| expect             | string       | expected output/return data           |
| disable            | bool         | if `true`, check will either not be started (or stopped, if already running) |
| tags               | string array | a set of any arbitrary tags that ease querying 9volt API or grouping checks together |
| warning-threshold  | int          | how many checks must fail before warning state |
| critical-threshold | int          | how many checks must fail before critical state |
| warning-alerter    | string array | if check enters warning state, the following alerters will be executed |
| critical-alerter   | string array | if check enters critical state, the following alerters will be executed |
| member-tag         | string       | require this check to only be assigned to members that are started/tagged w/ the same tag |

## Member Tag Details
The `member-tag` allows you to **require** that checks are assigned ONLY to nodes that match the same tag. This is helpful in cases where a remote service/host is ONLY accessible from a specific location (say due to network restrictions); another case would be if you are running `9volt` across a large WAN and want to be absolutely sure that a given service is available from multiple, different locations.

**Requiring geographic diversity example:**

* Launch 2 nodes in the US with **no** tags
* Launch 1 node in Spain with `spain` and `eu` tags
* Launch 1 node in Germany with `germany` and `eu` tags
* Launch 1 node in Japan with a `japan` tag
* In this type of a configuration:
  - Checks configured with **no** tags will be equally distributed among the 2 US nodes
  - Checks configured with an `eu` tag will be equally distributed among the nodes in Spain and Germany
  - Checks configured with a `japan` tag will be assigned to the node in Japan

**NOTE**: If a check is given a tag that does not have a corresponding node tag, that check will be **orphaned**, or in other words, it will not be assigned to any nodes until a node with that tag is started.

## Monitor Types

### Exec 
Execute a local `command` and expect it to complete with a `return-code`. You can also `expect` specific output from the command and/or specify a `timeout` that the command should complete by.

`command` expects a single string referencing the actual command/binary/script. If there are additional arguments, they should be defined as elements in an `args` array.

Example:

```yaml
monitor:
    exec-sample:
        type: exec
        description: an example exec check
        interval: 10s
        command: echo
        args:
            - hello
            - world
        expect: world
        return-code: 0
        timeout: 5s
```

|  Attribute  | Required |     Type     | Default | 
|-------------|----------|--------------|---------|
| type        | **true** | string       |    -    |
| command     | **true** | string       |    -    |
| interval    | **true** | duration     |    -    |
| return-code | false    | int          |    0    |
| args        | false    | string array |    -    |
| description | false    | string       |    -    |
| timeout     | false    | duration     |    3s   |
| expect      | false    | string       |    -    |

------------------------------------------

### HTTP 
Perform an `http` check against a target on port `80` expecting a `200` response status code.

You can further customize the check by specifying a custom port, ssl usage, path, method and expected response body content.

Example:

```yaml
monitor:
  example-http-check:
    type: http
    description: "Our special http check"
    host: cloudsy.com
    timeout: 5s
    interval: 10s
    port: 80
    status-code: 200
    expect: some words
    method: GET
    ssl: false
    url: /status/check
    tags:
      - my-team
      - golang
    warning-threshold: 1
    critical-threshold: 3
    warning-alerter:
      - primary-slack
    critical-alerter:
      - primary-slack
      - primary-pagerduty
```

|  Attribute  | Required |     Type     | Default | 
|-------------|----------|--------------|---------|
| type        | **true** | string       |    -    |
| host        | **true** | string       |    -    |
| interval    | **true** | duration     |    -    |
| description | false    | string       |    -    |
| timeout     | false    | duration     |    3s   |
| expect      | false    | string       |    -    |
| port        | false    | int          | 80 (443 if ssl == true) |
| status-code | false    | int          |   200   |
| method      | false    | string       |   GET   |
| ssl         | false    | bool         |  false  |
| url         | false    | string       |   ""    |

------------------------------------------

### TCP 
Perform a TCP connection check against a given host + port.

Further customize the check to `send` a custom payload, `expect` output, further tweak read/write timeouts and/or specify a larger/smaller read size.

NOTE: Many/most servers use carriage return to identify incoming bits of data. When using `send`, you may need to add a `\n` as part of your send string.

Example:

```yaml
monitor:
  ssh-expect-ssh-check:
    type: tcp
    description: "remote tcp check with expect"
    host: cloudsy.com
    timeout: 5s
    interval: 10s
    expect: OpenSSH
    port: 22
    tags:
      - team-core
      - golang
    warning-threshold: 1
    critical-threshold: 3
    warning-alerter:
      - secondary-slack
    critical-alerter:
      - secondary-slack
      - primary-pagerduty

```

|  Attribute  | Required |     Type     | Default | 
|-------------|----------|--------------|---------|
| type        | **true** | string       |    -    |
| host        | **true** | string       |    -    |
| interval    | **true** | duration     |    -    |
| port        | **true** | int          |    -    |
| description | false    | string       |    -    |
| timeout     | false    | duration     |    4s   |
| expect      | false    | string       |    -    |
| send        | false    | string       |    -    |
| read-timeout | false   | duration     |    2s   |
| write-timeout | false | duration      |    2s   |
| read-size   | false   | int           |  4096 (4K) |

### DNS
Perform various DNS resolution checks for a particular DNS query against a
specific DNS server.

This check can detect a number of possible errors with a DNS query. It attempts
to check the most important things first and only check the other settings
after those pass. It does not currently roll up a set of failures. The order
that checks proceed is:

1. Can we query this server at all?
2. If we had an expected number of records, did we get them?
3. Did we get any results?
4. Did we take longer than `dns-max-time` to resolve?
5. Did all of the records match the `expect` regex we passed?

Failing any of the above (if specified) will result in an error.

A sample configuration looks like:

```yaml
monitor:
  google-com-dns:
    type: dns
    description: "google.com DNS check"
    dns-target: "google.com"
    host: 8.8.8.8
    dns-expected-count: 5
    dns-record-type: "A"
    dns-max-time: "100ms"
    expect: "IN\\s+A"
    interval: 10s
    timeout: 1s
    warning-threshold: 1
    critical-threshold: 3
    warning-alerter:
      - primary-slack
    critical-alerter:
      - primary-email
```

We will make a DNS query to `host` and try to resolve the record `dns-target`.
Afterward we will make optional checks and then validate the regex contained in
`expect`. This will be matched against the full text line returned from DNS. E.g.
for an `A` record this would be matched against:

```
google.com.	299	IN	A	 209.85.202.102
```

|  Attribute         | Required |     Type     | Default   | 
|--------------------|----------|--------------|-----------|
| host               | **true** | string       |     -     |
| interval           | **true** | duration     |     -     |
| dns-target         | **true** | string       |     -     |
| expect             | false    | string(regex)|    "."    |
| dns-record-type    | false    | string       |    "A"    |
| dns-max-time       | false    | duration     |     -     |
| dns-expected-count | false    | int          |     -     |

