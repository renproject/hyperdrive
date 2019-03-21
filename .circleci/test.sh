# Simulate the environment used by Travis CI so that we can run local tests to
# find and resolve issues that are consistent with the Travis CI environment.
# This is helpful because Travis CI often finds issues that our own local tests
# do not.

# go vet ./...
# golint -set_exit_status `go list ./... | grep -Ev "(stackint/asm|vendor)"`
# golint `go list ./... | grep -Ev "(stackint/asm|vendor)"`

go build ./...
GOMAXPROCS=1 CI=true ginkgo -v --race --cover --coverprofile coverprofile.out ./...
covermerge 
  block/coverprofile.out          \
  cmd/hyperdrive/coverprofile.out \
  consensus/coverprofile.out      \
  replica/coverprofile.out        \
  shard/coverprofile.out          \
  sig/ecdsa/coverprofile.out      \
  sig/coverprofile.out            \
  supervisor/coverprofile.out     \
  tx/coverprofile.out > coverprofile.out
