vet:
	go vet -v -tags=sqlite_vtable ./...

build:
	go build -v -tags=sqlite_vtable askgit.go

xbuild:
	xgo -tags="sqlite_vtable" -targets="linux/386,linux/amd64,darwin/*" .
	ls askgit-* | xargs -I{} tar -czf "{}.tar.gz" "{}"
	shasum -a 512 askgit-*.tar.gz > checksums.txt

lint:
	golangci-lint run --build-tags sqlite_vtable

test:
	go test -v -tags=sqlite_vtable ./...

test-cover:
	go test -v -tags=sqlite_vtable ./... -cover -covermode=count -coverprofile=coverage.out
	go tool cover -html=coverage.out
