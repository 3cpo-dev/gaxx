#!/usr/bin/env bash
set -euo pipefail

KEY_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/gaxx/ssh"
mkdir -p "$KEY_DIR"
KEY_PATH="$KEY_DIR/id_ed25519"
if [[ -f "$KEY_PATH" ]];nthen
  echo "Key already exists at $KEY_PATH"
  exit 0
fi
ssh-keygen -t ed25519 -N "" -f "$KEY_PATH"
echo "Generated $KEY_PATH"


