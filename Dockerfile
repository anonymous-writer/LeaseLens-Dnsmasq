# Build the executable; application code lives in internal/app.
FROM golang:1.25-alpine AS build
WORKDIR /src

# Use Alpine's IEEE OUI registry at build time; only the text database is copied to runtime.
RUN apk add --no-cache arp-scan

COPY go.mod ./
COPY main.go ./
COPY internal/ ./internal/
COPY templates/ ./templates/
COPY static/ ./static/

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /dnsmasq-leases-ui .

FROM alpine:3.24
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app

ARG APP_VERSION=dev
ARG APP_RELEASE_DATE=
ARG REPO_URL=https://github.com/anonymous-writer/dnsmasq-leases-ui
ENV APP_VERSION=${APP_VERSION} \
    APP_RELEASE_DATE=${APP_RELEASE_DATE} \
    REPO_URL=${REPO_URL}

COPY --from=build /dnsmasq-leases-ui /app/dnsmasq-leases-ui
COPY --from=build /usr/share/arp-scan/ieee-oui.txt /usr/share/ieee-data/oui.txt

USER root
EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://127.0.0.1:${PORT:-5000}/leases >/dev/null || exit 1

ENTRYPOINT ["/app/dnsmasq-leases-ui"]
