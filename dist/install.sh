#!/usr/bin/env sh
set -e

# Colors (sin usar echo -e para evitar problemas con tar)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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
        exit 1
        ;;
esac

# === HEADER ===
printf "\n"
printf "${BLUE}╔════════════════════════════════════════╗${NC}\n"
printf "${BLUE}║    🦎 ChameleonDB Installer v0.1       ║${NC}\n"
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
    read -p "   Continue without PostgreSQL? (y/n): " -r
    if [ "$REPLY" != "y" ] && [ "$REPLY" != "Y" ]; then
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
    read -p "   Continue with reinstall? (y/n): " -r
    if [ "$REPLY" != "y" ] && [ "$REPLY" != "Y" ]; then
        printf "${BLUE}ℹ${NC} Installation cancelled\n"
        exit 0
    fi
fi

# === DOWNLOAD ===
URL="https://github.com/$REPO/releases/latest/download/chameleon-$OS-$ARCH.tar.gz"
printf "\n${BLUE}ℹ${NC} Downloading ChameleonDB...\n"
printf "   From: $URL\n"

TMP_DIR="$(mktemp -d)"
trap "rm -rf $TMP_DIR" EXIT

if ! curl -fsSL "$URL" -o "$TMP_DIR/chameleon.tar.gz"; then
    printf "${RED}✗${NC} Download failed\n"
    printf "   Check: https://github.com/$REPO/releases\n"
    exit 1
fi

printf "${GREEN}✓${NC} Downloaded successfully\n"

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
printf "${GREEN}✓${NC} ChameleonDB $VERSION installed successfully\n"

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
printf "  1. Initialize ChameleonDB into a new project:\n"
printf "     ${GREEN}cd my_project${NC}\n"
printf "     ${GREEN}chameleon init${NC}\n"
printf "\n"
printf "  2. Define your *.cham schemas in schema directory\n"
printf "\n"
printf "  3. Validate the schema:\n"
printf "     ${GREEN}chameleon validate${NC}\n"
printf "\n"
printf "  4. Generate migrations:\n"
printf "     ${GREEN}chameleon migrate --dry-run${NC}\n"
printf "\n"
printf "Documentation: https://chameleondb.dev/docs\n"
printf "GitHub:        https://github.com/$REPO\n"
printf "Discord:       https://chameleondb.dev/discord\n"
printf "\n"