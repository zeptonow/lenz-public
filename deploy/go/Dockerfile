# syntax=docker/dockerfile:1

FROM registry_url AS builder
ARG ACCESS_TOKEN
USER root
# Build with -tags musl so confluent-kafka-go statically links its bundled
# librdkafka. This avoids depending on the base image's librdkafka version,
# which on Alpine 3.21 is older than confluent-kafka-go's minimum requirement.
RUN apk add --no-cache --update make git openssh openssl-dev pkgconf gcc g++ libc-dev bash
USER cis
WORKDIR /app
ADD ./backend /app
RUN go mod download
RUN CGO_ENABLED=1 go build -buildvcs=false -tags musl -o openreplay .

FROM registry_url
WORKDIR /app

USER root
# librdkafka is bundled statically into the binary via -tags musl in the
# builder stage, so no system librdkafka is needed at runtime.
RUN apk update && apk add --no-cache \
    ca-certificates tzdata zstd ffmpeg

RUN wget -q https://raw.githubusercontent.com/ua-parser/uap-core/master/regexes.yaml -O /app/regexes.yaml && \
    wget -q https://raw.githubusercontent.com/geoip/geoip.mmdb -O /app/geoip.mmdb

RUN mkdir -p /mnt/efs

RUN mkdir -p /var/lib/lenz

USER cis

COPY --from=builder /app/openreplay ./
ENTRYPOINT ["./openreplay", "all"]
