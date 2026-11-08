#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../.." && pwd)"
CORE_DIR="$REPO_ROOT/chameleon-core"
CARGO_TOML="$CORE_DIR/Cargo.toml"

cd "$REPO_ROOT"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "El repositorio tiene cambios sin confirmar. Confirma o guarda esos cambios antes de crear un release." >&2
  exit 1
fi

current_version="$(awk '
  $0 == "[package]" { in_package = 1; next }
  in_package && /^version = "/ {
    sub(/^version = "/, "")
    sub(/"$/, "")
    print
    exit
  }
  in_package && /^\[/ { exit }
' "$CARGO_TOML")"

if [[ -z "$current_version" ]]; then
  echo "No se pudo encontrar la versión del paquete en $CARGO_TOML." >&2
  exit 1
fi

if [[ "$current_version" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(.*)$ ]]; then
  default_version="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.$((BASH_REMATCH[3] + 1))${BASH_REMATCH[4]}"
else
  echo "La versión actual no tiene un formato compatible con SemVer: $current_version" >&2
  exit 1
fi

echo "Versión actual: $current_version"
read -r -p "Nueva versión [v$default_version]: " requested_version
requested_version="${requested_version:-v$default_version}"
requested_version="${requested_version#v}"

if [[ ! "$requested_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.+][0-9A-Za-z.-]+)?$ ]]; then
  echo "Versión inválida: $requested_version" >&2
  exit 1
fi

if [[ "$requested_version" == "$current_version" ]]; then
  echo "La nueva versión es igual a la versión actual." >&2
  exit 1
fi

release_tag="v$requested_version"
component_tag="chameleon/$release_tag"

for tag in "$release_tag" "$component_tag"; do
  if git rev-parse --quiet --verify "refs/tags/$tag" >/dev/null; then
    echo "El tag ya existe: $tag" >&2
    exit 1
  fi
done

backup_dir="$(mktemp -d)"
cp "$CARGO_TOML" "$backup_dir/Cargo.toml"
restore_on_failure=1
cleanup() {
  status=$?
  if [[ "$status" -ne 0 && "$restore_on_failure" -eq 1 ]]; then
    cp "$backup_dir/Cargo.toml" "$CARGO_TOML"
    echo "Se restauró Cargo.toml porque el release no se pudo preparar." >&2
  fi
  rm -rf "$backup_dir"
  exit "$status"
}
trap cleanup EXIT

awk -v new_version="$requested_version" '
  $0 == "[package]" { in_package = 1 }
  in_package && !updated && /^version = "/ {
    print "version = \"" new_version "\""
    updated = 1
    next
  }
  in_package && /^\[/ && $0 != "[package]" { in_package = 0 }
  { print }
' "$CARGO_TOML" > "$backup_dir/Cargo.toml.new"
mv "$backup_dir/Cargo.toml.new" "$CARGO_TOML"

echo "Validando el manifiesto de Cargo..."
(cd "$CORE_DIR" && cargo check --quiet)

echo
echo "Cambio preparado:"
git diff -- "$CARGO_TOML"
echo

read -r -p "¿Crear el commit y los dos tags ahora? [Y/n]: " create_release
create_release="${create_release:-y}"

if [[ "$create_release" =~ ^[Yy]$ ]]; then
  git add "$CARGO_TOML"
  git commit -m "chore: release $release_tag" \
    -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
  restore_on_failure=0
  git tag "$release_tag"
  git tag "$component_tag"

  echo
  echo "Commit y tags creados:"
  echo "  $release_tag"
  echo "  $component_tag"
  echo
  read -r -p "¿Hacer push de la rama y los tags? [y/N]: " push_release
  if [[ "$push_release" =~ ^[Yy]$ ]]; then
    git push origin HEAD "$release_tag" "$component_tag"
  else
    echo "Para publicar el release:"
    echo "  git push origin HEAD $release_tag $component_tag"
  fi
else
  restore_on_failure=0
  echo
  echo "No se crearon commit ni tags. Para continuar:"
  echo "  git add chameleon-core/Cargo.toml"
  echo "  git commit -m 'chore: release $release_tag'"
  echo "  git tag $release_tag"
  echo "  git tag $component_tag"
  echo "  git push origin HEAD $release_tag $component_tag"
fi
