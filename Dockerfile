FROM golang:1.14 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -v -tags=sqlite_vtable askgit.go

FROM debian:buster-slim
WORKDIR /app/
RUN mkdir /repo
COPY --from=builder /app/askgit .

ENTRYPOINT ["./askgit", "--repo", "/repo"]

