FROM golang:1.21-alpine3.17 as development
WORKDIR /build
COPY . /build
RUN go build -o validation_relay \
    -ldflags "-X main.Commit=$COMMIT_HASH -X 'main.BuildTime=`date '+%b %_d %Y, %H:%M:%S'`'" \
    ./cmd/validation_relay
RUN go install github.com/mitranim/gow@latest
ENTRYPOINT ["gow", "run", "./cmd/validation_relay"]


FROM alpine:3.17
RUN apk add --no-cache ca-certificates
COPY --from=development /build/validation_relay /
ENTRYPOINT ["/validation_relay"]
