FROM golang:1.6-onbuild

CMD ./scripts/setup.sh --i-know-what-im-doing; go run main.go -e http://etcd:2379
