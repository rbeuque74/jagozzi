VERSION = `git describe --tags --dirty --always`
BINARY  = jagozzi

all:
	go build -race -o ${BINARY} -ldflags "-X main.version=${VERSION}" main.go utils.go instance.go

test:
	go test -race -timeout 10s -v ./...
