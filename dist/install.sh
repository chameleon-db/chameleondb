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
LIB_NAME_LINUX="libchameleon_core.so"
LIB_NAME_MACOS="libchameleon_core.dylib"
INSTALL_BIN_DIR="${CHAMELEON_INSTALL_DIR:-/usr/local/bin}"
INSTALL_LIB_DIR="${CHAMELEON_LIB_DIR:-/usr/local/lib}"

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
        ;;
    darwin) 
        OS="darwin"
        LIB_NAME="$LIB_NAME_MACOS"
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

if [ -w "$INSTALL_LIB_DIR" ]; then
    cp "$TMP_DIR/$LIB_NAME" "$INSTALL_LIB_DIR/$LIB_NAME"
    chmod 644 "$INSTALL_LIB_DIR/$LIB_NAME"
else
    sudo cp "$TMP_DIR/$LIB_NAME" "$INSTALL_LIB_DIR/$LIB_NAME"
    sudo chmod 644 "$INSTALL_LIB_DIR/$LIB_NAME"
fi

printf "${GREEN}✓${NC} Library installed\n"

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
printf "  1. Create a new project:\n"
printf "     ${GREEN}chameleon init my_project${NC}\n"
printf "\n"
printf "  2. Define your schema in schema.cham\n"
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