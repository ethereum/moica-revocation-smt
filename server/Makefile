.PHONY: build test test-integration proto clean

build:
	go build -o bin/smtserver ./cmd/smtserver

test:
	go test ./... -v

test-integration:
	go test -tags integration ./internal/crl/... -v -timeout 30m

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/revocation/revocation.proto

clean:
	rm -rf bin/

run: build
	./bin/smtserver
