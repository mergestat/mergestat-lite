FROM golang:1.20-bookworm@sha256:077ff85b374b23916b4b41835e242e5a3ddad9fc537ea7e980f230431747d245 as builder
WORKDIR /app
RUN apt-get update && apt-get -y install cmake libssl-dev libssl3 bzip2
COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
COPY git2go /app/git2go
RUN go mod download
COPY Makefile /app/Makefile
RUN make libgit2
COPY . .
RUN make
RUN ls -l

FROM debian:bookworm-slim@sha256:abbf1e0df2d9631707a41780bd9d332523d10cbb14560122536210298b77f09d
WORKDIR /app/
RUN mkdir /repo
COPY --from=builder /app/.build/mergestat .
RUN  /app/.build/mergestat
RUN apt-get update && apt-get install -y git

ENTRYPOINT ["./mergestat", "--repo", "/repo"]
