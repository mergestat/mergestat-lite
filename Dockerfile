FROM golang:1.19-buster as builder
WORKDIR /app
RUN apt-get update && apt-get -y install cmake libssl-dev
COPY . .
RUN make libgit2
RUN make

FROM debian:buster-slim
WORKDIR /app/
RUN mkdir /repo
COPY --from=builder /app/.build/mergestat .

RUN apt-get update && apt-get install -y git

ENTRYPOINT ["./mergestat", "--repo", "/repo"]
