#!/data/data/com.termux/files/usr/bin/bash
set -eu

export TRACE_BEGIN_SOONG=$(date +%s%N)
export TOP=$(builtin cd $(dirname ${BASH_SOURCE[0]})/../..; PWD= /data/data/com.termux/files/usr/bin/pwd)
builtin cd "${TOP}"
source "${TOP}/build/soong/scripts/microfactory.bash"
ulimit -a

soong_build_go multiproduct_kati android/soong/cmd/multiproduct_kati
exec "$(getoutdir)/multiproduct_kati" "$@"
