#!/usr/bin/env bash
set -euo pipefail

GOPROXY=off GOTOOLCHAIN=go1.26.5 \
  go test -race ./services/api/internal/storestatus -count=20 -timeout=15m
GOPROXY=off GOTOOLCHAIN=go1.26.5 \
  go test ./services/api/... -count=1 -timeout=15m
GOPROXY=off GOTOOLCHAIN=go1.26.5 \
  go test -race ./services/api/... -count=1 -timeout=15m
