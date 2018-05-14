#!/data/data/com.termux/files/usr/bin/bash -ex

if [ -z "${OUT_DIR}" ]; then
    echo Must set OUT_DIR
    exit 1
fi

TOP=$(pwd)

source build/envsetup.sh
PLATFORM_SDK_VERSION=$(get_build_var PLATFORM_SDK_VERSION)
PLATFORM_VERSION_ALL_CODENAMES=$(get_build_var PLATFORM_VERSION_ALL_CODENAMES)

PLATFORM_VERSION_ALL_CODENAMES=${PLATFORM_VERSION_ALL_CODENAMES/,/'","'}
PLATFORM_VERSION_ALL_CODENAMES="[\"${PLATFORM_VERSION_ALL_CODENAMES}\"]"

SOONG_OUT=${OUT_DIR}/soong
SOONG_NDK_OUT=${OUT_DIR}/soong/ndk
rm -rf ${SOONG_OUT}
mkdir -p ${SOONG_OUT}
cat > ${SOONG_OUT}/soong.config << EOF
{
    "Ndk_abis": true
}
EOF

# We only really need to set some of these variables, but soong won't merge this
# with the defaults, so we need to write out all the defaults with our values
# added.
cat > ${SOONG_OUT}/soong.variables << EOF
{
    "Platform_sdk_version": ${PLATFORM_SDK_VERSION},
    "Platform_version_active_codenames": ${PLATFORM_VERSION_ALL_CODENAMES},

    "DeviceName": "generic_arm64",
    "DeviceArch": "arm64",
    "DeviceArchVariant": "armv8-a",
    "DeviceCpuVariant": "denver64",
    "DeviceAbi": ["arm64-v8a"],
    "HostArch": "arm64",
    "Malloc_not_svelte": false,
    "Safestack": false
}
EOF
m --skip-make ${SOONG_OUT}/ndk.timestamp

if [ -n "${DIST_DIR}" ]; then
    mkdir -p ${DIST_DIR} || true
    tar cjf ${DIST_DIR}/ndk_platform.tar.bz2 -C ${SOONG_OUT} ndk
fi
