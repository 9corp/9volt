FROM golang:1.6-onbuild

CMD ./scripts/setup.sh; go run main.go -e http://etcd:2379
