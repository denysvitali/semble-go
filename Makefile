.PHONY: build test clean install run

BINARY=semble

build:
	go build -o $(BINARY) .

test:
	go test ./...

clean:
	rm -f $(BINARY)

install:
	go install .

run:
	go run . search "$(Q)" .
