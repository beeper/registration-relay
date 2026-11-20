FROM golang:1.27-alpine3.24 as builder
WORKDIR /build
COPY . /build
ENV GOTOOLCHAIN=local
RUN go build -o registration_relay \
    -ldflags "-X main.Commit=$COMMIT_HASH -X 'main.BuildTime=`date '+%b %_d %Y, %H:%M:%S'`'" \
    ./cmd/registration_relay

FROM builder as development
RUN go install github.com/mitranim/gow@latest
ENTRYPOINT ["gow", "run", "./cmd/registration_relay"]


FROM alpine:3.24
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/registration_relay /
ENTRYPOINT ["/registration_relay"]
