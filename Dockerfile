FROM golang:1.16.2-alpine3.12 as builder

RUN apk add --update git make

WORKDIR /workspace

COPY go.mod go.sum /workspace/
RUN go mod download

COPY . .

RUN make build_linux_amd64

FROM alpine:3.12.4

RUN apk add --update --no-cache \
    ca-certificates \
    git \
    openssh-client && \
  rm -rf /var/cache/apk/*

COPY --from=builder /workspace/dist/linux_amd64/baur /usr/local/bin/baur

ENTRYPOINT ["/usr/local/bin/baur"]
