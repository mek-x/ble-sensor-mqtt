FROM golang:1.15-alpine as golang
WORKDIR /go/src/app
COPY . .
# Static build required so that we can safely copy the binary over.
RUN CGO_ENABLED=0 GOARCH=arm go build -ldflags '-w -s'

FROM alpine:latest as alpine
RUN apk --no-cache add tzdata zip ca-certificates
WORKDIR /usr/share/zoneinfo
# -0 means no compression.  Needed because go's
# tz loader doesn't handle compressed data.
RUN zip -q -r -0 /zoneinfo.zip .

FROM scratch

# the test program:
COPY --from=golang /go/src/app/ble-sensor-mqtt /

# the timezone data:
ENV ZONEINFO /zoneinfo.zip
COPY --from=alpine /zoneinfo.zip /

# the tls certificates:
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/ble-sensor-mqtt"]
