#!/usr/bin/env bash

set -Eeuo pipefail
trap cleanup SIGINT SIGTERM ERR EXIT

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd -P)

tempfile=$(mktemp)
cleanup() {
    local exit_code=$?
    trap - SIGINT SIGTERM ERR EXIT
    rm -f ${tempfile}
    exit "$exit_code"
}

usage() {
	cat <<EOF
Usage: $(basename "${BASH_SOURCE[0]}") [-h] [-v] --schema PATH --out-dir PATH --package PKG

Processes qapi schema and generates go code.

Available options:

-h, --help            Print this help and exit
-v, --verbose         Print script debug info
--schema              Path to qapi schema
--out-dir             Path to an output folder
--package             Package name
EOF
	exit
}

SCHEMA=""
OUTDIR=""
PACKAGE=""
parse_params() {
	while :; do
		case "${1-}" in
		-h | --help) usage ;;
		-v | --verbose) set -x ;;
		--schema)
        SCHEMA="${2-}"
        shift
        ;;
		--out-dir)
        OUTDIR="${2-}"
        if [[ ! "${OUTDIR}" == /* ]]; then
            OUTDIR="$(pwd)/${OUTDIR}"
        fi
        shift
        ;;
		--package)
        PACKAGE="${2-}"
        shift
        ;;
		-?*) echo "Unknown option: $1" && exit 1 ;;
		*) break ;;
		esac
		shift
	done

	args=("$@")
    [ -z "${SCHEMA}" ] && echo "Missing parameter: --schema" && exit 1
    [ -z "${OUTDIR}" ] && echo "Missing parameter: --out-dir" && exit 1
    [ -z "${PACKAGE}" ] && echo "Missing parameter: --package" && exit 1

	return 0
}

parse_params "$@"

mkdir -p ${OUTDIR}

python3 -m venv ${script_dir}/.venv
source ${script_dir}/.venv/bin/activate
pip install jinja2 >/dev/null 2>&1

PYTHONPATH=${script_dir}/src/generator python3 ${script_dir}/qemu/scripts/qapi-gen.py -o ${OUTDIR} ${SCHEMA} --backend gobackend.QAPIGoBackend --prefix ${PACKAGE} >/dev/null
