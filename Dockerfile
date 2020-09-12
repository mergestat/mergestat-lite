FROM golang:1.14-buster as builder
WORKDIR /app
COPY scripts .
RUN apt-get update && apt-get -y install cmake libssl-dev
RUN ./install_libgit2.sh
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build

FROM debian:buster-slim
WORKDIR /app/
RUN mkdir /repo
COPY --from=builder /app/askgit .

RUN apt-get update && apt-get install -y git

ENTRYPOINT ["./askgit", "--repo", "/repo"]
