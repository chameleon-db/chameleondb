#!/usr/bin/env sh
set -e

# Colors (sin usar echo -e para evitar problemas con tar)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Ask a yes/no question. Reads from the terminal, so it also works when the
# script itself comes from a pipe (curl ... | sh). Without a terminal (CI) the
# default answer ($2) is used.
confirm() {
    if (exec < /dev/tty) 2>/dev/null; then
        printf "%s" "$1" > /dev/tty
        read -r REPLY < /dev/tty || REPLY="$2"
    else
        REPLY="$2"
    fi
    [ "$REPLY" = "y" ] || [ "$REPLY" = "Y" ]
}

# Configuración
REPO="chameleon-db/chameleondb"
BIN_NAME="chameleon"
LIB_NAME_LINUX="libchameleon.so"
LIB_NAME_MACOS="libchameleon.dylib"
HEADER_NAME="chameleon.h"
PKGCONFIG_NAME="chameleon.pc"
INSTALL_BIN_DIR="${CHAMELEON_INSTALL_DIR:-/usr/local/bin}"
INSTALL_LIB_DIR="${CHAMELEON_LIB_DIR:-/usr/local/lib}"
INSTALL_INCLUDE_DIR="${CHAMELEON_INCLUDE_DIR:-/usr/local/include}"
INSTALL_PKGCONFIG_DIR="${CHAMELEON_PKGCONFIG_DIR:-/usr/local/lib/pkgconfig}"

# Detectar OS
OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64) ARCH="arm64" ;;  # macOS M1/M2
    *)
        printf "${RED}✗${NC} Unsupported architecture: $ARCH\n"
        exit 1
        ;;
esac

case "$OS" in
    linux) 
        OS="linux"
        LIB_NAME="$LIB_NAME_LINUX"
        LIB_GLOB="libchameleon.so*"
        ;;
    darwin) 
        OS="darwin"
        LIB_NAME="$LIB_NAME_MACOS"
        LIB_GLOB="libchameleon*.dylib*"
        ;;
    *)
        printf "${RED}✗${NC} Unsupported OS: $OS\n"
        printf "   On Windows, download the archive from https://github.com/$REPO/releases\n"
        exit 1
        ;;
esac

# Prebuilt archives published with each release (see .github/workflows/release.yml)
case "$OS/$ARCH" in
    linux/amd64|linux/arm64) ASSET_NAME="chameleon-linux-$ARCH" ;;
    darwin/arm64)            ASSET_NAME="chameleon-macos-arm64" ;;
    darwin/amd64)
        printf "${RED}✗${NC} No prebuilt binary for Intel Macs (darwin/amd64)\n"
        printf "   Prebuilt macOS binaries are Apple Silicon only. Build from source:\n"
        printf "   https://github.com/$REPO#install-linux-and-macos\n"
        exit 1
        ;;
esac

# === HEADER ===
printf "\n"
printf "${BLUE}╔════════════════════════════════════════╗${NC}\n"
printf "${BLUE}║    🦎 ChameleonDB Installer            ║${NC}\n"
printf "${BLUE}║    Type-safe database access layer     ║${NC}\n"
printf "${BLUE}╚════════════════════════════════════════╝${NC}\n"
printf "\n"

# === DETECT ===
printf "${BLUE}ℹ${NC} Detected: ${OS}/${ARCH}\n"

# === CHECK PREREQUISITES ===
printf "${BLUE}ℹ${NC} Checking prerequisites...\n"

# Check PostgreSQL
if ! command -v psql >/dev/null 2>&1; then
    printf "${YELLOW}⚠${NC} PostgreSQL client (psql) not found\n"
    printf "   You'll need PostgreSQL to use ChameleonDB\n"
    printf "\n"
    printf "   Installation instructions:\n"
    printf "     Linux (Ubuntu):  sudo apt-get install postgresql-client\n"
    printf "     Linux (Fedora):  sudo dnf install postgresql\n"
    printf "     macOS:           brew install postgresql\n"
    printf "\n"
    if ! confirm "   Continue without PostgreSQL? (y/n): " y; then
        printf "${BLUE}ℹ${NC} Installation cancelled\n"
        exit 0
    fi
else
    printf "${GREEN}✓${NC} PostgreSQL client found\n"
fi

# Check curl
if ! command -v curl >/dev/null 2>&1; then
    printf "${RED}✗${NC} curl is required but not installed\n"
    exit 1
fi

# === CHECK IF ALREADY INSTALLED ===
if command -v chameleon >/dev/null 2>&1; then
    printf "${YELLOW}⚠${NC} ChameleonDB is already installed\n"
    printf "   Location: $(command -v chameleon)\n"
    if ! confirm "   Continue with reinstall? (y/n): " y; then
        printf "${BLUE}ℹ${NC} Installation cancelled\n"
        exit 0
    fi
fi

# === DOWNLOAD ===
# CHAMELEON_RELEASE_URL overrides the download base (mirrors, local testing).
BASE_URL="${CHAMELEON_RELEASE_URL:-https://github.com/$REPO/releases/latest/download}"
URL="$BASE_URL/$ASSET_NAME.tar.gz"
printf "\n${BLUE}ℹ${NC} Downloading ChameleonDB...\n"
printf "   From: $URL\n"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

if ! curl -fsSL "$URL" -o "$TMP_DIR/chameleon.tar.gz"; then
    printf "${RED}✗${NC} Download failed\n"
    printf "   No archive named $ASSET_NAME.tar.gz was found for this release.\n"
    printf "   Check: https://github.com/$REPO/releases\n"
    exit 1
fi

printf "${GREEN}✓${NC} Downloaded successfully\n"

# === VERIFY CHECKSUM ===
# Releases publish <asset>.sha256 next to each archive. A mismatch aborts the
# install. If the checksum file or a hashing tool is missing, we say so instead
# of silently skipping.
if command -v sha256sum >/dev/null 2>&1; then
    HASH_CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
    HASH_CMD="shasum -a 256"
else
    HASH_CMD=""
fi

if [ -z "$HASH_CMD" ]; then
    printf "${YELLOW}⚠${NC} No sha256sum/shasum found; checksum not verified\n"
elif curl -fsSL "$BASE_URL/$ASSET_NAME.sha256" -o "$TMP_DIR/expected.sha256" 2>/dev/null; then
    EXPECTED="$(awk '{print tolower($1); exit}' "$TMP_DIR/expected.sha256")"
    ACTUAL="$($HASH_CMD "$TMP_DIR/chameleon.tar.gz" | awk '{print tolower($1)}')"
    if [ -z "$EXPECTED" ] || [ "$EXPECTED" != "$ACTUAL" ]; then
        printf "${RED}✗${NC} Checksum mismatch; aborting\n"
        printf "   expected: $EXPECTED\n"
        printf "   actual:   $ACTUAL\n"
        exit 1
    fi
    printf "${GREEN}✓${NC} Checksum verified (sha256)\n"
else
    printf "${YELLOW}⚠${NC} No checksum published for this release; archive not verified\n"
fi

# === EXTRACT ===
printf "${BLUE}ℹ${NC} Extracting files...\n"

if ! tar -xzf "$TMP_DIR/chameleon.tar.gz" -C "$TMP_DIR"; then
    printf "${RED}✗${NC} Extraction failed\n"
    exit 1
fi

# === VERIFY FILES ===
if [ ! -f "$TMP_DIR/$BIN_NAME" ]; then
    printf "${RED}✗${NC} Binary not found in archive\n"
    exit 1
fi

if [ ! -f "$TMP_DIR/$LIB_NAME" ]; then
    printf "${RED}✗${NC} Library not found in archive\n"
    exit 1
fi

if [ ! -f "$TMP_DIR/$HEADER_NAME" ]; then
    printf "${YELLOW}⚠${NC} Header not found in archive (${HEADER_NAME})\n"
    printf "   SDK consumers (C/C++/bindings) may fail without it\n"
fi

printf "${GREEN}✓${NC} Files verified\n"

# === INSTALL BINARY ===
printf "\n${BLUE}ℹ${NC} Installing binary to $INSTALL_BIN_DIR\n"

if [ -w "$INSTALL_BIN_DIR" ]; then
    cp "$TMP_DIR/$BIN_NAME" "$INSTALL_BIN_DIR/$BIN_NAME"
    chmod +x "$INSTALL_BIN_DIR/$BIN_NAME"
else
    sudo cp "$TMP_DIR/$BIN_NAME" "$INSTALL_BIN_DIR/$BIN_NAME"
    sudo chmod +x "$INSTALL_BIN_DIR/$BIN_NAME"
fi

printf "${GREEN}✓${NC} Binary installed\n"

# === INSTALL LIBRARY ===
printf "${BLUE}ℹ${NC} Installing library to $INSTALL_LIB_DIR\n"

INSTALLED_LIBS=0

for LIB_SRC in "$TMP_DIR"/$LIB_GLOB; do
    [ -e "$LIB_SRC" ] || continue
    LIB_BASENAME="$(basename "$LIB_SRC")"
    LIB_DEST="$INSTALL_LIB_DIR/$LIB_BASENAME"

    if [ -w "$INSTALL_LIB_DIR" ]; then
        cp -P "$LIB_SRC" "$LIB_DEST"
        if [ ! -L "$LIB_DEST" ]; then
            chmod 644 "$LIB_DEST"
        fi
    else
        sudo cp -P "$LIB_SRC" "$LIB_DEST"
        if [ ! -L "$LIB_DEST" ]; then
            sudo chmod 644 "$LIB_DEST"
        fi
    fi

    INSTALLED_LIBS=$((INSTALLED_LIBS + 1))
done

if [ "$INSTALLED_LIBS" -eq 0 ]; then
    printf "${RED}✗${NC} No library files matched pattern: $LIB_GLOB\n"
    exit 1
fi

printf "${GREEN}✓${NC} Library installed\n"

# === INSTALL HEADER (SDK) ===
if [ -f "$TMP_DIR/$HEADER_NAME" ]; then
    printf "${BLUE}ℹ${NC} Installing header to $INSTALL_INCLUDE_DIR\n"

    if [ -w "$INSTALL_INCLUDE_DIR" ]; then
        cp "$TMP_DIR/$HEADER_NAME" "$INSTALL_INCLUDE_DIR/$HEADER_NAME"
        chmod 644 "$INSTALL_INCLUDE_DIR/$HEADER_NAME"
    else
        sudo cp "$TMP_DIR/$HEADER_NAME" "$INSTALL_INCLUDE_DIR/$HEADER_NAME"
        sudo chmod 644 "$INSTALL_INCLUDE_DIR/$HEADER_NAME"
    fi

    printf "${GREEN}✓${NC} Header installed\n"
fi

# === INSTALL PKG-CONFIG FILE (SDK) ===
printf "${BLUE}ℹ${NC} Installing pkg-config file to $INSTALL_PKGCONFIG_DIR\n"

TMP_PC_FILE="$TMP_DIR/$PKGCONFIG_NAME"
if [ ! -f "$TMP_PC_FILE" ]; then
    cat > "$TMP_PC_FILE" <<EOF
prefix=/usr/local
exec_prefix=\${prefix}
libdir=$INSTALL_LIB_DIR
includedir=$INSTALL_INCLUDE_DIR

Name: chameleon
Description: ChameleonDB core native library
Version: 0.0.0
Libs: -L$INSTALL_LIB_DIR -lchameleon
Cflags: -I$INSTALL_INCLUDE_DIR
EOF
fi

if [ ! -d "$INSTALL_PKGCONFIG_DIR" ]; then
    if [ -w "$(dirname "$INSTALL_PKGCONFIG_DIR")" ]; then
        mkdir -p "$INSTALL_PKGCONFIG_DIR"
    else
        sudo mkdir -p "$INSTALL_PKGCONFIG_DIR"
    fi
fi

if [ -w "$INSTALL_PKGCONFIG_DIR" ]; then
    cp "$TMP_PC_FILE" "$INSTALL_PKGCONFIG_DIR/$PKGCONFIG_NAME"
    chmod 644 "$INSTALL_PKGCONFIG_DIR/$PKGCONFIG_NAME"
else
    sudo cp "$TMP_PC_FILE" "$INSTALL_PKGCONFIG_DIR/$PKGCONFIG_NAME"
    sudo chmod 644 "$INSTALL_PKGCONFIG_DIR/$PKGCONFIG_NAME"
fi

printf "${GREEN}✓${NC} pkg-config file installed\n"

# === UPDATE LIBRARY CACHE ===
if command -v ldconfig >/dev/null 2>&1; then
    printf "${BLUE}ℹ${NC} Updating library cache\n"
    if [ -w "/etc/ld.so.cache" ]; then
        ldconfig
    else
        sudo ldconfig
    fi
fi

# === VERIFY INSTALLATION ===
printf "\n${BLUE}ℹ${NC} Verifying installation...\n"

if ! "$INSTALL_BIN_DIR/$BIN_NAME" version >/dev/null 2>&1; then
    printf "${RED}✗${NC} Installation verification failed\n"
    printf "   Try: $INSTALL_BIN_DIR/$BIN_NAME version\n"
    exit 1
fi

VERSION=$("$INSTALL_BIN_DIR/$BIN_NAME" version)
printf "${GREEN}✓${NC} $VERSION installed successfully\n"

# === VERIFY PATH ===
if ! command -v chameleon >/dev/null 2>&1; then
    printf "\n${YELLOW}⚠${NC} Add $INSTALL_BIN_DIR to your PATH\n"
    printf "   Add this to ~/.bashrc or ~/.zshrc:\n"
    printf "   export PATH=\"\\\$PATH:$INSTALL_BIN_DIR\"\n"
fi

# === NEXT STEPS ===
printf "\n"
printf "${BLUE}╔════════════════════════════════════════╗${NC}\n"
printf "${BLUE}║        Installation Complete!          ║${NC}\n"
printf "${BLUE}╚════════════════════════════════════════╝${NC}\n"
printf "\n"
printf "Next steps:\n"
printf "  1. Initialize ChameleonDB in a new project:\n"
printf "     ${GREEN}cd my_project${NC}\n"
printf "     ${GREEN}chameleon init${NC}\n"
printf "\n"
printf "  2. Edit your schema in schemas/ (init creates an example)\n"
printf "\n"
printf "  3. Validate the schema:\n"
printf "     ${GREEN}chameleon validate${NC}\n"
printf "\n"
printf "  4. Review the SQL, then apply it (set DATABASE_URL first):\n"
printf "     ${GREEN}chameleon migrate --dry-run${NC}\n"
printf "     ${GREEN}chameleon migrate --apply${NC}\n"
printf "\n"
printf "Documentation: https://docs.chameleondb.dev\n"
printf "Quickstart:    https://docs.chameleondb.dev/quickstart\n"
printf "GitHub:        https://github.com/$REPO\n"
printf "Discord:       https://chameleondb.dev/discord\n"
printf "\n"