#!/bin/sh
set -eu

XNODE_BIN="${XNODE_BIN:-xnode}"

"${XNODE_BIN}" --version

if [ -n "${XNODE_MOCK_PANEL:-}" ] &&
  [ -n "${NODE_ID:-}" ] &&
  [ -n "${NODE_DOMAIN:-}" ] &&
  [ -n "${DATA_DIR:-}" ] &&
  [ -n "${LOG_DIR:-}" ]; then
  "${XNODE_BIN}" --check
fi
