### Dev
- Assuming dev work is happening on OS X
- Run `bootstrap.sh`
    + This will install: `homebrew`, `golang`, `etcd` and various go packages
- If you see a "All looks well!", start `9volt` by doing the following:
    + `godep go run main.go -e http://localhost:2379`
- You should be good to go!

### Random musings
- Use `gofmt`
- Use the built-in race detector (`go run -race`)
- `foo.New()` should never error; treat as constructor
- `foo.Start()` for things that will complete (ie. may launch goroutines, etc.)
- `foo.Run()` for things that will *not* complete and should (probably) be launched in a goroutine
- Use bailout blocks/negative logic/negated if blocks when possible
- Make an effort to use interfaces - it will make testing *a lot* easier
- Using interfaces will allow you to use `counterfeiter` for generating fakes
    + https://github.com/maxbrunsfeld/counterfeiter
- At minimum, use an assertion library when writing tests (ie. https://github.com/stretchr/testify)
- Make an effort to ensure everything is unit testable first
- Tests should be tagged as follows (at the top of each test file):
    + For unit tests
        * `// +build unit`
    + For integration tests
        * `// +build integration`
    + For functional tests
        * `// +build functional`