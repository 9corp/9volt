Running 9volt in Docker
=======================

## Just get it up and running locally (using `docker-compose`)

This is the probably the *easiest* way to test out `9volt`; `docker-compose` will start up an instance of `etcd` and pull down the latest `9volt` image.

1. `docker-compose up -d`
2. `docker ps`
    * verify that `9volt` and `etcd` are running
3. Access `9volt` by opening up 

### But wait, there's more!

If you want to try out some of the cluster features, it's easy:

1. Update `docker-compose.yml` to include another `9volt` section, except this time specify a different listen address.
```
# Your docker-compose.yml file would look something like this
version: '2'
services:
  9volt:
    depends_on:
      - etcd
    image: 9corp/9volt:latest
    ports:
      - "8080:8080"
    links:
     - etcd
    environment:
      - NINEV_ETCD_MEMBERS=http://etcd:2379
  9volt2:
    depends_on:
      - etcd
    image: 9corp/9volt:latest
    ports:
      - "8181:8181"
    links:
     - etcd
    environment:
      - NINEV_ETCD_MEMBERS=http://etcd:2379
      - NINEV_LISTEN_ADDRESS=:8181
  etcd:
    image: quay.io/coreos/etcd:v3.1.2
    ports:
      - "2379:2379"
    command: /usr/local/bin/etcd -advertise-client-urls http://0.0.0.0:2379 -listen-client-urls http://0.0.0.0:2379
```
2. Start up `docker-compose up -d`
3. Visit http://localhost:8080/ui/Cluster and see the second node!
4. Bonus: Add additional sections to `docker-compose.yml` to expand the cluster further!

## Or build your own image (and verify that it works)

1. Make sure `make`, `golang` and `nodejs` are available locally
2. Build a `9volt` docker image
    * `make installtools`
    * `make build/docker`
3. Run the container
    * `docker run -d -p 8080:8080 -e NINEV_ETCD_MEMBERS=http://etcd-host1:2379  9volt:a9e86c3`
        * `a9e86c3` is the short git sha/version of this image (that is set during `make build/docker`)

Once `9volt` has started, you should be able to do something along the lines of this to verify all is well:

```
fullstop:9volt dselans$ docker ps | grep 9volt
c14b7aa6af53        9volt:a9e86c3             "/9volt-linux server"    About a minute ago   Up About a minute   0.0.0.0:8080->8080/tcp                                                    peaceful_pasteur
fullstop:9volt dselans$ docker logs c14b7aa6af53
time="2017-03-01T07:29:01Z" level=info msg="cluster-directorMonitor: Current director '844f039a' expired; time to upscale!"
time="2017-03-01T07:29:01Z" level=info msg="cluster-directorMonitor: Taking over director role"
time="2017-03-01T07:29:01Z" level=info msg="manager: Starting manager components..."
time="2017-03-01T07:29:01Z" level=info msg="alerter: Starting alerter components..."
time="2017-03-01T07:29:01Z" level=info msg="director-stateListener: Starting up etcd watchers"
time="2017-03-01T07:29:01Z" level=info msg="state: Starting state components..."
time="2017-03-01T07:29:01Z" level=info msg="9volt has started! API address: http://0.0.0.0:8080 MemberID: 5cc497ad"
time="2017-03-01T07:29:01Z" level=info msg="ui: statik mode (from statik.go)"
time="2017-03-01T07:29:04Z" level=info msg="director-distributeChecks: Performing check distribution across members in cluster"
time="2017-03-01T07:29:04Z" level=error msg="director-stateListener: Unable to (re)distribute checks: Check configuration is empty - nothing to distribute!"
```

If `docker ps` does not have `9volt`, `9volt` was not able to start; check the associated container logs (ie. `docker ps -a | grep 9volt`, find the container id and `docker logs container_id`).
