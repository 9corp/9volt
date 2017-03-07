9volt startup env vars
======================

You can pass several config env vars to `9volt` at startup, which will allow you to not have to pass `-e` (etcd-members) or other things like `tags` via CLI flags.

| Env Var | Default | Description | Example |
|---------|---------|-------------|---------|
| `NINEV_ETCD_MEMBERS` | N/A | Comma separated list of etcd member URL's | `export NINEV_ETCD_MEMBERS="http://localhost:2379,http://some.etcd.host.example.com:2379"` |
| `NINEV_LISTEN_ADDRESS` | 0.0.0.0:8080 | Listen address that 9volt should bind to | `$ export NINEV_LISTEN_ADDRESS=":8181"` |
| `NINEV_ETCD_PREFIX` | 9volt | What prefix 9volt should use when reading/writing to/from etcd | `$ export NINEV_ETCD_PREFIX="my-prefix"` |
| `NINEV_MEMBER_TAGS` | N/A | Comma separated list of tags (this can allow you to (re)distribute and assign checks to nodes matching specific tags. See more info [here](MONITOR_CONFIGS.md)) | `$ export NINEV_MEMBER_TAGS="tag1,tag2"` |

**NOTE**: Command line params override env vars. Ie. In `export NINEV_MEMBER_TAGS="one two three"; ./9volt -e http://localhost:2379 -t "1 2 3"` - `tags` will be set to `1`, `2` and `3`. Env vars *will* override defaults however.
