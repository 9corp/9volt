# Monitor Documentation

"Special" types:

**duration**: a string that contains a decimal number and a unit suffix for 'ns', 'us', 'ms', 's', 'm' and 'h'. Read more [here](https://golang.org/pkg/time/#ParseDuration).

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
