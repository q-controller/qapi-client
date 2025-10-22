#!/usr/bin/env bash

set Eu
trap cleanup SIGINT SIGTERM ERR EXIT

QEMU_LOGS=/tmp/qemu.logs
cleanup() {
    trap - SIGINT SIGTERM ERR EXIT
    # script cleanup here
    cat ${QEMU_LOGS}
    rm -fr ${QEMU_LOGS}
}

VERSION=24.04
IMAGE=ubuntu-${VERSION}-server-cloudimg-amd64.img
if [ ! -f "./${IMAGE}" ]; then
    curl -LO https://cloud-images.ubuntu.com/releases/${VERSION}/release/${IMAGE}
fi

ACCEL="kvm"
if [ "$(uname)" == "Darwin" ]; then
    ACCEL="hvf"
fi

ARCH=$(uname -m)
if [ "$ARCH" == "arm64" ] || [ "$ARCH" == "aarch64" ]; then
    QEMU_BIN="qemu-system-aarch64"
    MACHINE_OPTS="-machine virt -cpu cortex-a57"
else
    QEMU_BIN="qemu-system-x86_64"
    MACHINE_OPTS="-machine q35"
fi

${QEMU_BIN} \
    ${MACHINE_OPTS} -accel ${ACCEL} -m 2048 -nographic \
    -hda ./${IMAGE} \
    -nographic \
    -qmp "unix:/tmp/example.qmp,server,wait=off" > ${QEMU_LOGS} 2>&1
