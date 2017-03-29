
# 


Table of Contents

1. [Cluster State](#cluster)
1. [Fetch event data (optionally filtered by one or more event types)](#event)
1. [Monitor Configuration](#monitor)
1. [Fetch check state data including latest check status, ownership, last check timestamp;](#state)

<a name="cluster"></a>

## cluster

| Specification | Value |
|-----|-----|
| Resource Path | /cluster |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes |  |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| /cluster | [GET](#Fetch Cluster Stats) | Fetches cluster state data (membership, current director, heartbeats) |



<a name="Fetch Cluster Stats"></a>

#### API: /cluster (GET)


Fetches cluster state data (membership, current director, heartbeats)



| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | object | [ClusterStats](#github.com.9corp.9volt.dal.ClusterStats) |  |
| 500 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |




### Models

<a name="encoding.json.RawMessage"></a>

#### RawMessage

| Field Name (alphabetical) | Field Type | Description |
|-----|-----|-----|

<a name="github.com.9corp.9volt.dal.ClusterStats"></a>

#### ClusterStats

| Field Name (alphabetical) | Field Type | Description |
|-----|-----|-----|
| Director | encoding.json.RawMessage |  |
| Members | array |  |

<a name="github.com.InVisionApp.rye.JSONStatus"></a>

#### JSONStatus

| Field Name (alphabetical) | Field Type | Description |
|-----|-----|-----|
| message | string |  |
| status | string |  |


<a name="event"></a>

## event

| Specification | Value |
|-----|-----|
| Resource Path | /event |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes | application/json |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| /event | [GET](#Fetch Events) | Fetch event data (optionally filtered by one or more event types) |



<a name="Fetch Events"></a>

#### API: /event (GET)


Fetch event data (optionally filtered by one or more event types)



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| type | query | string | comma separated event types |  |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | array | [Event](#github.com.9corp.9volt.event.Event) |  |
| 500 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |




### Models

<a name="github.com.9corp.9volt.event.Event"></a>

#### Event

| Field Name (alphabetical) | Field Type | Description |
|-----|-----|-----|
| memberid | string |  |
| message | string |  |
| timestamp | Time |  |
| type | string |  |

<a name="github.com.InVisionApp.rye.JSONStatus"></a>

#### JSONStatus

| Field Name (alphabetical) | Field Type | Description |
|-----|-----|-----|
| message | string |  |
| status | string |  |


<a name="monitor"></a>

## monitor

| Specification | Value |
|-----|-----|
| Resource Path | /monitor |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes | application/json |
| Produces |  |



### Operations


| Resource Path | Operation | Description |
|-----|-----|-----|
| /monitor | [POST](#Add Monitor Configuration) | Add/Update monitor config |
| /monitor/\{check\} | [GET](#Fetch Monitor Configuration) | Fetch all (or specific) monitor configuration(s) from etcd |
| /monitor/\{check\} | [GET](#Set Disabled State for Given Monitor) | Enable or disable a specific monitor configuration (changes are immediate) |
| /monitor/\{check\} | [DELETE](#Delete existing monitor configuration) | Delete monitor config |


<a name="Add Monitor Configuration"></a>

#### API: /monitor (POST)

Add/Update monitor config

| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| N/A | POST | object (map[string][MonitorConfig](https://github.com/9corp/9volt/blob/master/docs/MONITOR_CONFIGS.md)) | Collection of monitors to add | Yes |


Example payload:
```json
{
	"exec-check1": {
		  "type": "exec",
		  "description": "exec check test",
		  "timeout": "5s",
		  "command": "echo",
		  "args": [
		    "hello",
		    "world"
		  ],
		  "interval": "10s",
		  "return-code": 0,
		  "expect": "world",
		  "warning-threshold": 1,
		  "critical-threshold": 3,
		  "warning-alerter": [
		    "primary-slack"
		  ],
		  "critical-alerter": [
		    "primary-email"
		  ],
		  "tags": [
		    "dbservers",
		    "mysql"
		]
	}
}
```


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | array | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |
| 400 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |
| 500 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |

<a name="Fetch Monitor Configuration"></a>

#### API: /monitor/\{check\} (GET)


Fetch all (or specific) monitor configuration(s) from etcd



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| check | path | string | Specific check name |  |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | array | [fullMonitorConfig](#github.com.9corp.9volt.api.fullMonitorConfig) |  |
| 500 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |


<a name="Set Disabled State for Given Monitor"></a>

#### API: /monitor/\{check\} (GET)


Enable or disable a specific monitor configuration (changes are immediate)



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| check | path | string | Specific check name | Yes |
| disable | query | string | Disable/enable a check |  |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | array | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |
| 500 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |


<a name="Delete existing monitor configuration"></a>

#### API: /monitor/\{check\} (DELETE)

Delete existing monitor config

| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| check | path | string | Specific check name | Yes |

| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | array | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |
| 404 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |
| 500 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |


### Models

<a name="github.com.9corp.9volt.api.fullMonitorConfig"></a>

#### fullMonitorConfig

| Field Name (alphabetical) | Field Type | Description |
|-----|-----|-----|

<a name="github.com.InVisionApp.rye.JSONStatus"></a>

#### JSONStatus

| Field Name (alphabetical) | Field Type | Description |
|-----|-----|-----|
| message | string |  |
| status | string |  |


<a name="state"></a>

## alerter

| Specification | Value |
|-----|-----|
| Resource Path | /alerter |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes | application/json |
| Produces |  |


### Operations

| Resource Path | Operation | Description |
|-----|-----|-----|
| /alerter | [POST](#Add Alerter Configuration) | Add/Update alerter config |
| /alerter/\{alerterName\} | [GET](#Fetch Alerter Configuration) | Fetch all (or specific) alerter configuration(s) from etcd |
| /alerter/\{alerterName\} | [DELETE](#Delete existing alerter configuration) | Delete alerter config |

<a name="Add Alerter Configuration"></a>

#### API: /alerter (POST)

Add/Update alerter config

| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| N/A | POST | object (map[string][AlerterConfig](https://github.com/9corp/9volt/blob/master/docs/ALERTER_CONFIGS.md)) | Collection of alerters to add | Yes |


Example payload:
```json
{
	"test-alerter": {
		"type": "slack",
		"description": "foobar",
		"options": {
			"channel": "some-team",
			"icon-url": "https://pbs.twimg.com/profile_images/593893225045200896/r9uL4jWU.png",
			"token": "asdfasdfasdfasdfasdf",
			"username": "monit"
		}
	}
}
```


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | array | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |
| 400 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |
| 500 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |


<a name="Fetch Alerter Configuration"></a>

#### API: /alerter/\{alerterName\} (GET)


Fetch all (or specific) alerter configuration(s) from etcd


| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| alerterName | path | string | Specific alerter name |  |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | array | [fullAlerterConfig](#github.com.9corp.9volt.api.fullAlerterConfig) |  |
| 500 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |

<a name="Delete existing alerter configuration"></a>

#### API: /alerter/\{alerterName\} (DELETE)

Delete existing alerter config

| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| alerterName | path | string | Specific check name | Yes |

| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | array | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |
| 404 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |
| 500 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |

### Models

<a name="github.com.9corp.9volt.api.fullAlerterConfig"></a>

#### fullAlerterConfig

| Field Name (alphabetical) | Field Type | Description |
|-----|-----|-----|

## state

| Specification | Value |
|-----|-----|
| Resource Path | /state |
| API Version |  |
| BasePath for the API | {{.}} |
| Consumes | application/json |
| Produces |  |



### Operations

| Resource Path | Operation | Description |
|-----|-----|-----|
| /state | [GET](#Fetch Check State Data) | Fetch check state data including latest check status, ownership, last check timestamp; |



<a name="Fetch Check State Data"></a>

#### API: /state (GET)


Fetch check state data including latest check status, ownership, last check timestamp;



| Param Name | Param Type | Data Type | Description | Required? |
|-----|-----|-----|-----|-----|
| tags | query | string | One or more tags (comma separated) |  |


| Code | Type | Model | Message |
|-----|-----|-----|-----|
| 200 | array | [Message](#github.com.9corp.9volt.state.Message) |  |
| 500 | object | [JSONStatus](#github.com.InVisionApp.rye.JSONStatus) |  |




### Models

<a name="encoding.json.RawMessage"></a>

#### RawMessage

| Field Name (alphabetical) | Field Type | Description |
|-----|-----|-----|

<a name="github.com.9corp.9volt.state.Message"></a>

#### Message

| Field Name (alphabetical) | Field Type | Description |
|-----|-----|-----|
| check | string |  |
| config | encoding.json.RawMessage |  |
| count | int |  |
| date | Time |  |
| message | string |  |
| owner | string |  |
| status | string |  |

<a name="github.com.InVisionApp.rye.JSONStatus"></a>

#### JSONStatus

| Field Name (alphabetical) | Field Type | Description |
|-----|-----|-----|
| message | string |  |
| status | string |  |


