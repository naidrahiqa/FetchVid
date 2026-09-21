#!/bin/bash
export PATH="$HOME/go/bin:$PATH"
export PKG_CONFIG_PATH="$HOME/.local/lib/pkgconfig:${PKG_CONFIG_PATH:-}"
cd "$(dirname "$0")"
exec wails dev "$@"
