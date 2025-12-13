#!/bin/sh
set -e

if command -v systemctl >/dev/null 2>&1; then
  systemctl stop gridbeat.service || true
  systemctl disable gridbeat.service || true
  systemctl daemon-reload || true
fi

exit 0