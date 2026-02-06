#!/usr/bin/env sh
set -e

REPO="chameleon-db/chameleondb"
BIN_NAME="chameleon"
LIB_NAME="libchameleon_core.so"
INSTALL_BIN_DIR="/usr/local/bin"
INSTALL_LIB_DIR="/usr/local/lib"

OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  *)
    echo "❌ Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux) 
    OS="linux"
    LIB_EXT="so"
    ;;
  darwin) 
    OS="darwin"
    LIB_EXT="dylib"
    ;;
  *)
    echo "❌ Unsupported OS: $OS"
    exit 1
    ;;
esac

# Currently only Linux is supported
if [ "$OS" != "linux" ]; then
  echo "❌ Only Linux is currently supported"
  echo "   macOS and Windows support coming soon"
  echo ""
  echo "   To build from source:"
  echo "   git clone https://github.com/$REPO.git"
  echo "   cd chameleondb && make build"
  exit 1
fi

URL="https://github.com/$REPO/releases/latest/download/chameleon-$OS-$ARCH.tar.gz"

echo "🦎 Installing ChameleonDB ($OS/$ARCH)..."
echo "📦 Downloading from: $URL"

TMP_DIR="$(mktemp -d)"
trap "rm -rf $TMP_DIR" EXIT

# Download
if ! curl -fsSL "$URL" -o "$TMP_DIR/chameleon.tar.gz"; then
  echo "❌ Download failed. Is the latest release available?"
  echo "   Check: https://github.com/$REPO/releases"
  exit 1
fi

# Extract
tar -xzf "$TMP_DIR/chameleon.tar.gz" -C "$TMP_DIR"

# Check files exist
if [ ! -f "$TMP_DIR/$BIN_NAME" ]; then
  echo "❌ Binary not found in archive"
  exit 1
fi

if [ ! -f "$TMP_DIR/$LIB_NAME" ]; then
  echo "❌ Library not found in archive"
  exit 1
fi

# Install binary
echo "📥 Installing binary to $INSTALL_BIN_DIR (may require sudo)"
if [ -w "$INSTALL_BIN_DIR" ]; then
  cp "$TMP_DIR/$BIN_NAME" "$INSTALL_BIN_DIR/$BIN_NAME"
  chmod +x "$INSTALL_BIN_DIR/$BIN_NAME"
else
  sudo cp "$TMP_DIR/$BIN_NAME" "$INSTALL_BIN_DIR/$BIN_NAME"
  sudo chmod +x "$INSTALL_BIN_DIR/$BIN_NAME"
fi

# Install library
echo "📥 Installing library to $INSTALL_LIB_DIR"
if [ -w "$INSTALL_LIB_DIR" ]; then
  cp "$TMP_DIR/$LIB_NAME" "$INSTALL_LIB_DIR/$LIB_NAME"
  chmod 644 "$INSTALL_LIB_DIR/$LIB_NAME"
else
  sudo cp "$TMP_DIR/$LIB_NAME" "$INSTALL_LIB_DIR/$LIB_NAME"
  sudo chmod 644 "$INSTALL_LIB_DIR/$LIB_NAME"
fi

# Update library cache
if command -v ldconfig >/dev/null 2>&1; then
  echo "🔄 Updating library cache"
  if [ -w "/etc/ld.so.cache" ]; then
    ldconfig
  else
    sudo ldconfig
  fi
fi

echo ""
echo "✅ Installation complete!"
echo ""
echo "   Verify: chameleon version"
echo "   Get started: chameleon init myproject"
echo ""
echo "   Documentation: https://chameleondb.dev/docs"
echo "   Examples: https://github.com/chameleon-db/chameleon-examples"