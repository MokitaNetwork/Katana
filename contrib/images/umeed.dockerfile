# Stage-1: build
# We use Debian Bullseye rather then Alpine because Alpine has problem building libwasmvm
# - requires to download libwasmvm_muslc from external source. Build with glibc is straightforward.
FROM golang:1.19-bullseye AS builder

WORKDIR /src/
COPY . .

RUN LEDGER_ENABLED=false BUILD_TAGS=badgerdb make install

# Stage-2: copy binary and required artifacts to a fresh image
# we need to use debian compatible system.
FROM ubuntu:rolling
# RUN apt update && apt upgrade -y ca-certificates

COPY --from=builder /go/bin/katanad /usr/local/bin/
COPY --from=builder /go/pkg/mod/github.com/\!cosm\!wasm/wasmvm\@v*/internal/api/libwasmvm.*.so /usr/lib/

EXPOSE 26656 26657 1317 9090

# Run katanad by default, omit entrypoint to ease using container with CLI
CMD ["katanad"]
STOPSIGNAL SIGTERM
