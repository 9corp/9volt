### Etcd layout notes

- /9volt/config (key)
- /9volt/cluster (dir)
- /9volt/cluster/members (dir)
    + contains (ttl'd) cluster member blobs
- /9volt/cluster/members/member_id (key)
    + cluster members update their own blob
    + **[When director]** cluster engine monitors this dir, informs director when a member has dropped out
    + `{"id":"asdf1234", "host":"hostname", "listen_address" : "0.0.0.0:8080", "last_updated": "time.Now()"}`
- /9volt/cluster/director (key)
    + current director blob and their latest heartbeat
    + **[When NOT director]** cluster members monitor this key to ensure director is alive and well
- /9volt/monitor (dir)
    + contains monitoring config blobs
- /9volt/host (dir)
    + contains host config blobs
- /9volt/alert (dir)
    + contains alert config blobs