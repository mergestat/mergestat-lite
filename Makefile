vet:
	go vet -v -tags=sqlite_vtable ./...

build:
	go build -v -tags=sqlite_vtable gitqlite.go

lint:
	golangci-lint run --build-tags sqlite_vtable

test:
	go test -v -tags=sqlite_vtable ./...
