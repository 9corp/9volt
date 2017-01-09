# Alerter Config Documentation

Alerter configs are needed if you want to actually alert a monitor state change; they are the glue to monitoring configs.

## Alerter Types

### Slack
The Slack alerter will post formatted warning/critical message to a given channel. It can optionally use a custom icon and username.

Example: 
```yaml
alerter:
  secondary-slack:
    type: slack
    description: "secondary slack alerter"
    options:
      token: bar
      channel: 9volt-testing
      username: robibi2
      icon-url: http://cdn.akamai.steamstatic.com/steamcommunity/public/images/avatars/d2/d25fd479e446f3bef884cbedb5b2b643133b93fc_full.jpg
```

|  Attribute  | Required |  Type  | Default | 
|-------------|----------|--------|---------|
| type        | **true** | string |    -    |
| options ->  | **true** |   -    |    -    |
| token       | **true** | string |    -    |
| channel     | **true** | string |    -    |
| description | false    | string |    -    |
| icon-url    | false    | string | "default slack bot icon" |
| username    | false    | string | "username configured for bot in slack" | 

### Pagerduty
A no-thrills pagerduty alerter -- will open (and resolve) incidents as the monitor changes state between warning <-> critical <-> OK.

Example: 
```yaml
alerter:
  primary-pagerduty:
    type: pagerduty
    description: "primary pagerduty alerter"
    options:
      token: bar
```

|  Attribute  | Required |  Type  | Default | 
|-------------|----------|--------|---------|
| type        | **true** | string |    -    |
| options ->  | **true** |   -    |    -    |
| token       | **true** | string |    -    |
| description | false    | string |    -    |

### Email
Email alerter allows you to send an email to a destination email address. You can optionally configure outbound smtp server authentication.

NOTE 1: If sending via an SMTP server, the `address` must be in the format of `host:port`.
NOTE 2: If authentication is required, `auth` must be set to either `plain` or `md5`. `username` and `password` is also required if `auth` is enabled.
NOTE 3: If your email is not going through, turn on debug logging for 9volt (or look at the events via the API).

Example: 
```yaml
alerter:
  primary-email:
    type: email
    description: "primary email alerter"
    options:
      to: daniel.selans@gmail.com
      address: smtp.gmail.com:587
      username: user
      password: pass
      auth: plain | md5

```

|  Attribute  | Required |  Type  | Default | 
|-------------|----------|--------|---------|
| type        | **true** | string |    -    |
| options ->  | **true** |   -    |    -    |
| to          | **true** | string |    -    |
| from        | false    | string |    -    |
| address     | false    | string |    -    |
| username*   | false    | string |    -    |
| password*   | false    | string |    -    |
| auth*       | false    | string |    -    |
| description | false    | string |    -    |

* If `auth` is enabled, `username` and `password` is required as well.
