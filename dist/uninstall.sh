#!/usr/bin/env sh
set -e

BIN="/usr/local/bin/chameleon"
LIB="/usr/local/lib/chameleon"
CONF="$HOME/.chameleon"

echo ""
echo "ChameleonDB Uninstaller"
echo "-----------------------"
echo ""
echo "This will remove:"
[ -f "$BIN" ] && echo " - Binary: $BIN"
[ -d "$LIB" ] && echo " - Libraries: $LIB"

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

# Remove libraries only if directory exists and is empty-safe
if [ -d "$LIB" ]; then
  if [ "$(ls -A "$LIB")" ]; then
    echo "Warning: $LIB is not empty. Skipping removal."
  else
    echo "Removing libraries..."
    sudo rmdir "$LIB"
  fi
fi

echo ""
echo "✔ ChameleonDB uninstalled successfully"
echo "ℹ User data and projects were preserved"