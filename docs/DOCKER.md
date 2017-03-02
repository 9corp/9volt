Running 9volt in Docker
=======================

At some point, official images will be pushed to some public docker repo. For now, that's not the case, so you'll have to build `9volt` and the images yourself - which thankfully is not very difficult.

All build processes depend on `make`, `golang` and `nodejs`. For OS X users, install via brew: `brew install golang && brew install nodejs`. For other OS's, consult your favorite package manager.

Golang 1.6+ should be fine.

### Just get it up and running locally (using `docker-compose`)

To avoid having large containers^, our `Dockerfile` (and by proxy `docker-compose.yml`) assume that the latest build is in `./build/`. Thus, we first need to perform an actual build. There's a `make` target for that though, so easy peasy.

1. Run `make installtools`
2. Run `make build/docker-compose`
3. Access `9volt` at `http://localhost:8080/`

^ _While the Dockerfile *could* build the whole thing, our build process requires nodejs (for building the UI) and we would rather avoid bloating the image (600MB w/ nodejs VS ~30MB with just the 9volt bin)._

### Or no fuss, just build an image (and try to create/start a container)

1. Build a `9volt` docker image
    * `make installtools`
    * `make build/docker`
2. Run the container
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
