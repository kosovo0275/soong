#!/data/data/com.termux/files/usr/bin/bash
set -eu

export TRACE_BEGIN_SOONG=$(date +%s%N)

gettop() {
    local TOPFILE=build/soong/root.bp
    if [ -z "${TOP-}" -a -f "${TOP-}/${TOPFILE}" ] ; then
        (builtin cd $TOP; PWD= /data/data/com.termux/files/usr/bin/pwd)
    else
        if [ -f $TOPFILE ] ; then
            PWD= /data/data/com.termux/files/usr/bin/pwd
        else
            local HERE=$PWD
            T=
            while [ \( ! \( -f $TOPFILE \) \) -a \( $PWD != "/" \) ]; do
                builtin cd ..
                T=`PWD= /data/data/com.termux/files/usr/bin/pwd -P`
            done
            builtin cd $HERE
            if [ -f "$T/$TOPFILE" ]; then
                echo $T
            fi
        fi
    fi
}

export ORIGINAL_PWD=${PWD}
export TOP=$(gettop)
source ${TOP}/build/soong/scripts/microfactory.bash

soong_build_go soong_ui android/soong/cmd/soong_ui

builtin cd ${TOP}
exec "$(getoutdir)/soong_ui" "$@"
