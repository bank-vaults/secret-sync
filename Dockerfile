FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.4.0@sha256:0cd3f05c72d6c9b038eb135f91376ee1169ef3a330d34e418e65e2a5c2e9c0d4 AS xx

FROM --platform=$BUILDPLATFORM golang:1.22-alpine3.18@sha256:d995eb689a0c123590a3d34c65f57f3a118bda3fa26f92da5e089ae7d8fd81a0 AS builder

COPY --from=xx / /

RUN apk add --update --no-cache ca-certificates make git curl clang lld

ARG TARGETPLATFORM

RUN xx-apk --update --no-cache add musl-dev gcc

RUN xx-go --wrap

WORKDIR /usr/local/src/secret-sync

ARG GOPROXY

ENV CGO_ENABLED=0

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -o /usr/local/bin/secret-sync .
RUN xx-verify /usr/local/bin/secret-sync


FROM alpine:3.19.1@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b

RUN apk add --update --no-cache ca-certificates tzdata

COPY --from=builder /usr/local/bin/secret-sync /usr/local/bin/secret-sync

USER 65534
