### Etcd layout notes

- /9volt/config (key)
- /9volt/cluster (dir)
- /9volt/cluster/members (dir)
    + contains (ttl'd) cluster member blobs
- /9volt/cluster/members/member_id (dir)
    + contains `status` JSON blob
        * `"{\"MemberID\":\"365c6d2c\",\"Hostname\":\"fullstop.local\",\"ListenAddress\":\"0.0.0.0:8080\",\"LastUpdated\":\"2016-07-04T18:30:10.905842389-07:00\"}",`
    + cluster members update their own blob
    + **[When director]** cluster engine monitors this dir, informs director when a member has dropped out
    + contains `config` dir, which contains references to the actual monitor config
        * `/9volt/cluster/members/member_id/config/base64_key`, value = `/9volt/monitor/check_name`
- /9volt/cluster/director (key)
    + current director blob and their latest heartbeat
    + **[When NOT director]** cluster members monitor this key to ensure director is alive and well
- /9volt/monitor (dir)
    + contains monitoring config blobs
- /9volt/alert (dir)
    + contains alert config blobs
