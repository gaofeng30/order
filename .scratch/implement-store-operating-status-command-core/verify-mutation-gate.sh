#!/usr/bin/env bash
set -euo pipefail

mutation_gate_repo=$(git rev-parse --show-toplevel)
exec bash "${mutation_gate_repo}/.scratch/implement-store-operating-status-command-core/verify-mysql.sh" \
  bash "${mutation_gate_repo}/.scratch/implement-store-operating-status-command-core/verify-mutations.sh"
