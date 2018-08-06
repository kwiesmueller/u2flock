default: install

install:
	go build -o $(GOPATH)/bin/u2flock cmd/*.go