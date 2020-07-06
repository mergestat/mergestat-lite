FROM golang:1.14 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -v -tags=sqlite_vtable gitqlite.go

FROM debian:buster-slim
WORKDIR /app/
RUN mkdir /repo
COPY --from=builder /app/gitqlite .

ENTRYPOINT ["./gitqlite", "--repo", "/repo"]

