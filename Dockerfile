ARG GO_IMAGE=golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2
ARG BASE_IMAGE=scratch

FROM ${GO_IMAGE} AS builder
RUN apk add --no-cache ca-certificates make
WORKDIR /opt/gobetween
COPY src/go.mod src/go.sum ./src/
COPY go.mod go.sum ./
RUN go mod download && cd src && go mod download
COPY . .
ARG VERSION=0.8.2
ARG REVISION=unknown
ARG BRANCH=unknown
RUN make build-static VERSION="${VERSION}" REVISION="${REVISION}" BRANCH="${BRANCH}" \
    && chmod 0755 bin/gobetween

FROM ${BASE_IMAGE}
ARG VERSION=0.8.2
ARG REVISION=unknown
WORKDIR /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /opt/gobetween/bin/gobetween /gobetween
# Numeric non-root default; OpenShift may replace this with its namespace UID.
# No writable log directory or pidfile is needed with the chart configuration.
USER 65532:0
EXPOSE 4000/udp 9284/tcp
STOPSIGNAL SIGTERM
ENTRYPOINT ["/gobetween"]
CMD ["-c", "/etc/gobetween/conf/gobetween.toml"]
LABEL org.opencontainers.image.title="gobetween UDP" \
      org.opencontainers.image.source="https://github.com/yyyar/gobetween" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}"
