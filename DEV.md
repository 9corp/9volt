### Dev
- Assuming dev work is happening on OS X
- Run `bootstrap.sh`
    + This will install: `homebrew`, `golang`, `etcd` and various go packages
- If you see a "All looks well!", start `9volt` by doing the following:
    + `godep go run main.go -e http://localhost:2379`
- You should be good to go!

### Random musings
- Make an effort to use interfaces - it will make testing *a lot* easier
- Speaking of testing - use `counterfeiter` to generate fakes
    + https://github.com/maxbrunsfeld/counterfeiter
- Use `gofmt`
- Make an effort to ensure everything is unit testable
