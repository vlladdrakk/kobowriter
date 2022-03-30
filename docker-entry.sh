#!/usr/bin/env bash

set -e
source /home/ubuntu/koxtoolchain/refs/x-compile.sh kobo env
export PATH="$PATH:/home/ubuntu/go/bin"
export GOPATH="/opt/go"

exec "$@"
