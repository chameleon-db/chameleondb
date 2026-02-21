#!/usr/bin/env sh
set -e

BIN="/usr/local/bin/chameleon"
LIB_DIR="/usr/local/lib"
HEADER="/usr/local/include/chameleon.h"
PKGCONFIG="/usr/local/lib/pkgconfig/chameleon.pc"
CONF="$HOME/.chameleon"

echo ""
echo "ChameleonDB Uninstaller"
echo "-----------------------"
echo ""
echo "This will remove:"
[ -f "$BIN" ] && echo " - Binary: $BIN"
for lib in "$LIB_DIR"/libchameleon.so* "$LIB_DIR"/libchameleon*.dylib* "$LIB_DIR"/chameleon.dll; do
  [ -e "$lib" ] && echo " - Library: $lib"
done
[ -f "$HEADER" ] && echo " - Header: $HEADER"
[ -f "$PKGCONFIG" ] && echo " - pkg-config: $PKGCONFIG"

echo ""
echo "The following will NOT be removed:"
echo " - Project files"
echo " - Databases"
echo " - User configuration (~/.chameleon)"
echo ""

printf "Do you want to continue? [y/N]: "
read ans

case "$ans" in
  y|Y|yes|YES)
    echo ""
    ;;
  *)
    echo "Uninstall cancelled."
    exit 0
    ;;
esac

# Remove binary
if [ -f "$BIN" ]; then
  echo "Removing binary..."
  sudo rm "$BIN"
fi

# Remove native libraries (versioned + symlinks)
for lib in "$LIB_DIR"/libchameleon.so* "$LIB_DIR"/libchameleon*.dylib* "$LIB_DIR"/chameleon.dll; do
  [ -e "$lib" ] || continue
  echo "Removing library ($lib)..."
  sudo rm "$lib"
done

# Remove header
if [ -f "$HEADER" ]; then
  echo "Removing header..."
  sudo rm "$HEADER"
fi

# Remove pkg-config metadata
if [ -f "$PKGCONFIG" ]; then
  echo "Removing pkg-config metadata..."
  sudo rm "$PKGCONFIG"
fi

echo ""
echo "✔ ChameleonDB uninstalled successfully"
echo "ℹ User data and projects were preserved"