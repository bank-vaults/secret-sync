FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.2.1@sha256:8879a398dedf0aadaacfbd332b29ff2f84bc39ae6d4e9c0a1109db27ac5ba012 AS xx

FROM --platform=$BUILDPLATFORM golang:1.21.1-alpine3.18@sha256:ec31b7fbaf32583af4f4f8c9e0cabc651247624bf962adbb7fa1b54c9c03d5e4 AS builder

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


FROM alpine:3.18.3@sha256:7144f7bab3d4c2648d7e59409f15ec52a18006a128c733fcff20d3a4a54ba44a

RUN apk add --update --no-cache ca-certificates tzdata

COPY --from=builder /usr/local/bin/secret-sync /usr/local/bin/secret-sync

USER 65534
