# Fetch base packages
FROM golang:1.19-alpine AS base-builder
ENV PACKAGES make git libc-dev gcc linux-headers ca-certificates build-base
RUN apk add --no-cache $PACKAGES

# Fetch base katana packages
FROM base-builder AS katana-base-builder
ENV PACKAGES curl bash eudev-dev python3
RUN apk add --no-cache $PACKAGES

# Compile the katanad binary
FROM katana-base-builder AS katanad-builder
WORKDIR /src/katana
COPY . .
RUN go mod download

# Cosmwasm - Download correct libwasmvm version
RUN WASMVM_VERSION=$(go list -m github.com/CosmWasm/wasmvm | cut -d ' ' -f 2) && \
    wget https://github.com/CosmWasm/wasmvm/releases/download/$WASMVM_VERSION/libwasmvm_muslc.$(uname -m).a \
      -O /lib/libwasmvm_muslc.a && \
    # verify checksum
    wget https://github.com/CosmWasm/wasmvm/releases/download/$WASMVM_VERSION/checksums.txt -O /tmp/checksums.txt && \
    sha256sum /lib/libwasmvm_muslc.a | grep $(cat /tmp/checksums.txt | grep $(uname -m) | cut -d ' ' -f 1)

RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make install
RUN cd price-feeder && LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make install

# # Fetch peggo (gravity bridge) binary
FROM base-builder AS peggo-builder
ARG PEGGO_VERSION=v0.3.0
WORKDIR /src/peggo
RUN git clone https://github.com/umee-network/peggo.git
RUN cd peggo && git checkout ${PEGGO_VERSION} && make build && cp ./build/peggo /usr/local/bin/

# Add to a distroless container
FROM gcr.io/distroless/cc:debug
COPY --from=katanad-builder /go/bin/* /usr/local/bin/
COPY --from=peggo-builder /usr/local/bin/peggo /usr/local/bin/

EXPOSE 26656 26657 1317 9090 7171
ENTRYPOINT ["katanad", "start"]
