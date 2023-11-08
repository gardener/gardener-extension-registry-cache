############# builder
FROM golang:1.21.4 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-registry-cache
COPY . .
RUN make install

############# base
FROM gcr.io/distroless/static-debian12:nonroot AS base
WORKDIR /

############# gardener-extension-registry-cache
FROM base AS registry-cache

COPY --from=builder /go/bin/gardener-extension-registry-cache /gardener-extension-registry-cache
ENTRYPOINT ["/gardener-extension-registry-cache"]

############# gardener-extension-registry-cache-admission
FROM base AS registry-cache-admission

COPY --from=builder /go/bin/gardener-extension-registry-cache-admission /gardener-extension-registry-cache-admission
ENTRYPOINT ["/gardener-extension-registry-cache-admission"]
