ci:
	! gofmt -l *.go | read nothing
	go vet
	go test -v ./...
	go get golang.org/x/lint/golint
	golint *.go
