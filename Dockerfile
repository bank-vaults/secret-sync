FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.2.1@sha256:8879a398dedf0aadaacfbd332b29ff2f84bc39ae6d4e9c0a1109db27ac5ba012 AS xx

FROM --platform=$BUILDPLATFORM golang:1.21.2-alpine3.18@sha256:a76f153cff6a59112777c071b0cde1b6e4691ddc7f172be424228da1bfb7bbda AS builder

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


FROM alpine:3.18.4@sha256:eece025e432126ce23f223450a0326fbebde39cdf496a85d8c016293fc851978

RUN apk add --update --no-cache ca-certificates tzdata

COPY --from=builder /usr/local/bin/secret-sync /usr/local/bin/secret-sync

USER 65534
