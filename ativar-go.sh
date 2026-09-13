#!/bin/bash

export PROJETO_GO="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export GOROOT="$PROJETO_GO/.local/go1.26"
export GOPATH="$PROJETO_GO/.local/gopath"
export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"

echo " Ambiente Go do VAIJUNTO ativado"
echo "GOROOT: $GOROOT"
echo "GOPATH: $GOPATH"
echo
go version
echo
