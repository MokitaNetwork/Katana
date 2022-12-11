#!/bin/bash -eu

# Download the katanad binary with the same version as mainnet and unpack it

# USAGE: ./download-mainnet-katanad.sh

is_macos() {
  [[ "$OSTYPE" == "darwin"* ]]
}

architecture=$(uname -m)

CWD="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
MAINNET_VERSION=${MAINNET_VERSION:-"v3.0.2"}

download_mainnet_binary(){
  # Checks for the katanad v1 file
  if [ ! -f "$KATANAD_BIN_MAINNET" ]; then
    echo "$KATANAD_BIN_MAINNET doesn't exist"

    if [ -z $KATANAD_BIN_MAINNET_URL_TARBALL ]; then
      echo You need to set the KATANAD_BIN_MAINNET_URL_TARBALL variable
      exit 1
    fi

    KATANAD_RELEASES_PATH=$CWD/katanad-releases
    mkdir -p $KATANAD_RELEASES_PATH
    wget -c $KATANAD_BIN_MAINNET_URL_TARBALL -O - | tar -xz -C $KATANAD_RELEASES_PATH

    KATANAD_BIN_MAINNET_BASENAME=$(basename $KATANAD_BIN_MAINNET_URL_TARBALL .tar.gz)
    KATANAD_BIN_MAINNET=$KATANAD_RELEASES_PATH/$KATANAD_BIN_MAINNET_BASENAME/katanad
  fi
}

mac_mainnet() {
  if [[ "$architecture" == "arm64" ]];then
    KATANAD_BIN_MAINNET_URL_TARBALL=${KATANAD_BIN_MAINNET_URL_TARBALL:-"https://github.com/mokitanetwork/katana/releases/download/${MAINNET_VERSION}/katanad-${MAINNET_VERSION}-darwin-arm64.tar.gz"}
    KATANAD_BIN_MAINNET=${KATANAD_BIN_MAINNET:-"$CWD/katanad-releases/katanad-${MAINNET_VERSION}-darwin-arm64/katanad"}
  else
    KATANAD_BIN_MAINNET_URL_TARBALL=${KATANAD_BIN_MAINNET_URL_TARBALL:-"https://github.com/mokitanetwork/katana/releases/download/${MAINNET_VERSION}/katanad-${MAINNET_VERSION}-darwin-amd64.tar.gz"}
    KATANAD_BIN_MAINNET=${KATANAD_BIN_MAINNET:-"$CWD/katanad-releases/katanad-${MAINNET_VERSION}-darwin-amd64/katanad"}
  fi
}

linux_mainnet(){
  if [[ "$architecture" == "arm64" ]];then
    KATANAD_BIN_MAINNET_URL_TARBALL=${KATANAD_BIN_MAINNET_URL_TARBALL:-"https://github.com/mokitanetwork/katana/releases/download/${MAINNET_VERSION}/katanad-${MAINNET_VERSION}-linux-arm64.tar.gz"}
    KATANAD_BIN_MAINNET=${KATANAD_BIN_MAINNET:-"$CWD/katanad-releases/katanad-${MAINNET_VERSION}-linux-arm64/katanad"}
  else
    KATANAD_BIN_MAINNET_URL_TARBALL=${KATANAD_BIN_MAINNET_URL_TARBALL:-"https://github.com/mokitanetwork/katana/releases/download/${MAINNET_VERSION}/katanad-${MAINNET_VERSION}-linux-amd64.tar.gz"}
    KATANAD_BIN_MAINNET=${KATANAD_BIN_MAINNET:-"$CWD/katanad-releases/katanad-${MAINNET_VERSION}-linux-amd64/katanad"}
  fi
}

if is_macos ; then
  mac_mainnet
  download_mainnet_binary
else
  linux_mainnet
  download_mainnet_binary
fi
