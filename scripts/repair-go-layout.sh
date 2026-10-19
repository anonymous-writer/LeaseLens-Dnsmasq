#!/usr/bin/env bash
set -euo pipefail

# Run this from the root of the leaselens-dnsmasq Git repository, after
# extracting this ZIP over the checkout so internal/app/ is updated.
repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "ERROR: Bitte im Root-Verzeichnis eines Git-Checkouts ausführen." >&2
  exit 1
}
cd "$repo_root"

if [[ ! -f internal/app/app.go || ! -f main.go ]]; then
  echo "ERROR: Die neue Projektstruktur fehlt. Bitte zuerst das komplette ZIP in den Checkout extrahieren." >&2
  exit 1
fi

# The modular layout keeps only main.go in the repository root. Legacy Go
# source files at root conflict with package main and must not be compiled.
shopt -s nullglob
removed=()
for file in ./*.go; do
  [[ "${file#./}" == "main.go" ]] && continue
  rm -- "$file"
  removed+=("${file#./}")
done

if ((${#removed[@]})); then
  printf 'Entfernte veraltete Go-Dateien im Projekt-Root:\n'
  printf '  - %s\n' "${removed[@]}"
else
  echo "Keine veralteten Go-Dateien im Projekt-Root gefunden."
fi

printf '\nProjektstruktur geprüft. Folgende Befehle bitte danach ausführen:\n'
printf '  go test ./...\n  go vet ./...\n  git status --short\n'
printf '\nÄnderungen anschließend committen und pushen.\n'
