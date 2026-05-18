#!/usr/bin/env bash
#
# Wrap `gobl-tbai send` with the Bizkaia sandbox cert and license that ship
# in test/certs. Usage:
#
#     ./send-test.sh path/to/signed-envelope.json
#
# Any extra flags after the filename are forwarded to the CLI (e.g. --prev).
#
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <signed-envelope.json> [extra flags]" >&2
  exit 64
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${REPO_ROOT}/bin/gobl-tbai"

mkdir -p "${REPO_ROOT}/bin"
(cd "${REPO_ROOT}" && go build -o "${BIN}" ./cmd/gobl.ticketbai)

INPUT="$1"
shift

"${BIN}" send "${INPUT}" \
  --cert            "${REPO_ROOT}/test/certs/EntitateOrdezkaria_RepresentanteDeEntidad.p12" \
  --password        "IZDesa2025" \
  --sw-nif          "A99800005" \
  --sw-company-name "SOFTWARE GARANTE TICKETBAI PRUEBA" \
  --sw-name         "Invopop" \
  --sw-version      "1.0" \
  --sw-license      "TBAIBI00000000PRUEBA" \
  "$@"
