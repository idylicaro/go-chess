#!/bin/bash

# SPRT of the working tree's engine against an earlier commit (#43).
#
# Usage: scripts/sprt.sh [base-ref] [-- extra flags]
#
#   scripts/sprt.sh                    # working tree vs HEAD, SPRT [0, 10]
#   scripts/sprt.sh main -- --tc 5+0.05 --sprt 0,5
#   RUNNER=cutechess OPENINGS=book.pgn scripts/sprt.sh main
#
# The base engine is built from base-ref in a temporary git worktree and the
# new one from the working tree. By default the match runs with
# `chess match`; RUNNER=cutechess uses cutechess-cli instead, which needs
# OPENINGS to name a PGN or EPD file. Extra flags go to the runner.

set -euo pipefail

BASE_REF="${1:-HEAD}"
shift || true
if [ "${1:-}" = "--" ]; then
	shift
fi

ROOT="$(git rev-parse --show-toplevel)"
WORK="$(mktemp -d)"
trap 'git -C "$ROOT" worktree remove --force "$WORK/base" >/dev/null 2>&1 || true; rm -rf "$WORK"' EXIT

echo "Building the base engine from $BASE_REF and the new one from the working tree..."
git -C "$ROOT" worktree add --detach "$WORK/base" "$BASE_REF" >/dev/null
(cd "$WORK/base" && go build -o "$WORK/engine-base" ./cmd/engine)
(cd "$ROOT" && go build -o "$WORK/engine-new" ./cmd/engine && go build -o "$WORK/chess" ./cmd/cli)

CONCURRENCY="${CONCURRENCY:-$(($(nproc) / 2))}"

if [ "${RUNNER:-match}" = "cutechess" ]; then
	if [ -z "${OPENINGS:-}" ]; then
		echo "RUNNER=cutechess needs OPENINGS=<file.pgn|file.epd>" >&2
		exit 2
	fi
	format="${OPENINGS##*.}"
	cutechess-cli \
		-engine cmd="$WORK/engine-new" name=new \
		-engine cmd="$WORK/engine-base" name=base \
		-each proto=uci tc=10+0.1 \
		-openings file="$OPENINGS" format="$format" order=random \
		-games 2 -repeat -rounds 5000 -concurrency "$CONCURRENCY" \
		-sprt elo0=0 elo1=10 alpha=0.05 beta=0.05 \
		-ratinginterval 20 -recover "$@"
	exit
fi

"$WORK/chess" match \
	--engine1 "$WORK/engine-new" --name1 new \
	--engine2 "$WORK/engine-base" --name2 base \
	--sprt 0,10 --concurrency "$CONCURRENCY" --pairs 5000 "$@"
