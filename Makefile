VERSION = `git describe --tags --dirty --always`
BINARY  = jagozzi

all:
	go build -o ${BINARY} -ldflags "-X main.version=${VERSION}" main.go utils.go instance.go

test:
	go test -timeout 10s -v ./...

test-coverage:
	go test -timeout 10s -v -covermode=count -coverprofile=coverage.out ./...

gometalinter:
	gometalinter --enable-all --disable=lll
