#!/usr/bin/env bash

APP_NAME="beekeeper"
REPO_URL="https://github.com/ethersphere/beekeeper"

: "${USE_SUDO:="true"}"
: "${BEEKEEPER_INSTALL_DIR:="/usr/local/bin"}"

detect_arch() {
  ARCH=$(uname -m)
  case $ARCH in
    armv5*) ARCH="armv5";;
    armv6*) ARCH="armv6";;
    armv7*) ARCH="arm";;
    aarch64) ARCH="arm64";;
    x86) ARCH="386";;
    x86_64) ARCH="amd64";;
    i686) ARCH="386";;
    i386) ARCH="386";;
  esac
}

detect_os() {
  OS=$(uname|tr '[:upper:]' '[:lower:]')

  case "$OS" in
    # Minimalist GNU for Windows
    mingw*) OS='windows';;
  esac
}

run_as_root() {
  local CMD="$*"

  if [ $EUID -ne 0 ] && [ $USE_SUDO = "true" ]; then
    CMD="sudo $CMD"
  fi

  $CMD
}

supported() {
  local supported="darwin-amd64\nlinux-386\nlinux-amd64\nlinux-arm64\nlinux-armv6"
  if ! echo "${supported}" | grep -q "${OS}-${ARCH}"; then
    if [ $OS == "windows" ]; then
      echo "Auto install not supported for Windows."
      echo "Install binary from here $REPO_URL/releases"
      exit 1
    else
      echo "No prebuilt binary for ${OS}-${ARCH}."
      echo "To build from source, go to $REPO_URL"
      exit 1
    fi
  fi

  if ! command -v curl &> /dev/null && ! command -v wget &> /dev/null; then
    echo "Either curl or wget is required"
    exit 1
  fi
}

# check_installed_version checks which version of beekeeper is installed and
# if it needs to be changed.
check_installed_version() {
  if [[ -f "${BEEKEEPER_INSTALL_DIR}/${APP_NAME}" ]]; then
    local version=$(beekeeper version 2>&1)
    if [[ "${version%-*}" == "${TAG#v}" ]]; then
      echo "beekeeper ${version} is already ${DESIRED_VERSION:-latest}"
      return 0
    else
      echo "beekeeper ${TAG} is available. Changing from version ${version}."
      return 1
    fi
  else
    return 1
  fi
}

# check_tag_provided checks whether TAG has provided as an environment variable so we can skip check_latest_version.
check_tag_provided() {
  [[ ! -z "$TAG" ]]
}

# check_latest_version grabs the latest version string from the releases
check_latest_version() {
  local latest_release_url="$REPO_URL/releases/latest"
  if command -v curl &> /dev/null; then
    TAG=$(curl -Ls -o /dev/null -w %{url_effective} $latest_release_url | grep -oE "[^/]+$" )
  elif command -v wget &> /dev/null; then
    TAG=$(wget $latest_release_url --server-response -O /dev/null 2>&1 | awk '/^  Location: /{DEST=$2} END{ print DEST}' | grep -oE "[^/]+$")
  fi
}

# download_file downloads the latest binary package and also the checksum
# for that binary.
download_file() {
  BEEKEEPER_DIST="beekeeper-$OS-$ARCH"
  if [ "$OS" == "windows" ]; then
    BEEKEEPER_DIST="beekeeper-$OS-$ARCH.exe"
  fi
  DOWNLOAD_URL="$REPO_URL/releases/download/$TAG/$BEEKEEPER_DIST"
  BEEKEEPER_TMP_ROOT="$(mktemp -dt beekeeper-binary-XXXXXX)"
  BEEKEEPER_TMP_FILE="$BEEKEEPER_TMP_ROOT/$BEEKEEPER_DIST"
  if command -v curl &> /dev/null; then
    curl -SsL "$DOWNLOAD_URL" -o "$BEEKEEPER_TMP_FILE"
  elif command -v wget &> /dev/null; then
    wget -q -O "$BEEKEEPER_TMP_FILE" "$DOWNLOAD_URL"
  fi
}

# install_file verifies the SHA256 for the file, then unpacks and
# installs it.
install_file() {
  echo "Preparing to install $APP_NAME into ${BEEKEEPER_INSTALL_DIR}"
  run_as_root chmod +x "$BEEKEEPER_TMP_FILE"
  run_as_root cp "$BEEKEEPER_TMP_FILE" "$BEEKEEPER_INSTALL_DIR/$APP_NAME"
  echo "$APP_NAME installed into $BEEKEEPER_INSTALL_DIR/$APP_NAME"
}

# fail_trap is executed if an error occurs.
fail_trap() {
  result=$?
  if [ "$result" != "0" ]; then
    if [[ -n "$INPUT_ARGUMENTS" ]]; then
      echo "Failed to install $APP_NAME with the arguments provided: $INPUT_ARGUMENTS"
      help
    else
      echo "Failed to install $APP_NAME"
    fi
    echo -e "\tFor support, go to $REPO_URL."
  fi
  cleanup
  exit $result
}

# test_binary tests the installed client to make sure it is working.
test_binary() {
  if ! command -v $APP_NAME &> /dev/null; then
    echo "$APP_NAME not found. Is $BEEKEEPER_INSTALL_DIR on your "'$PATH?'
    exit 1
  fi
  echo "Run '$APP_NAME --help' to see what you can do with it."
}

# help provides possible cli installation arguments
help () {
  echo "Accepted cli arguments are:"
  echo -e "\t[--help|-h] ->> prints this help"
  echo -e "\t[--no-sudo]  ->> install without sudo"
}

# cleanup temporary files
cleanup() {
  if [[ -d "${BEEKEEPER_TMP_ROOT:-}" ]]; then
    rm -rf "$BEEKEEPER_TMP_ROOT"
  fi
}

# Execution

#Stop execution on any error
trap "fail_trap" EXIT
set -e

# Parsing input arguments (if any)
export INPUT_ARGUMENTS="${@}"
set -u
while [[ $# -gt 0 ]]; do
  case $1 in
    '--no-sudo')
       USE_SUDO="false"
       ;;
    '--help'|-h)
       help
       exit 0
       ;;
    *) exit 1
       ;;
  esac
  shift
done
set +u

detect_arch
detect_os
supported
check_tag_provided || check_latest_version
if ! check_installed_version; then
  download_file
  install_file
fi

test_binary
cleanup
