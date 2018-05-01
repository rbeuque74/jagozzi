all:
	go build -o jagozzi main.go utils.go instance.go

test:
	go test -v ./...
