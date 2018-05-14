#!/data/data/com.termux/files/usr/bin/bash

prog="$0"
while [ -h "${prog}" ]; do
    fullprog=`/data/data/com.termux/files/usr/bin/ls -ld "${prog}"`
    fullprog=`expr "${fullprog}" : ".* -> \(.*\)$"`
    if expr "x${fullprog}" : 'x/' >/dev/null; then
        prog="${fullprog}"
    else
        progdir=`dirname "${prog}"`
        prog="${progdir}/${fullprog}"
    fi
done

oldwd=`pwd`
progdir=`dirname "${prog}"`
builtin cd "${progdir}"
progdir=`pwd`
prog="${progdir}"/`basename "${prog}"`
builtin cd "${oldwd}"

jarfile=`basename "${prog}"`.jar
jardir="${progdir}"

if [ ! -r "${jardir}/${jarfile}" ]; then
    jardir=`dirname "${progdir}"`/framework
fi

if [ ! -r "${jardir}/${jarfile}" ]; then
    echo `basename "${prog}"`": can't find ${jarfile}"
    exit 1
fi

javaOpts=""
while expr "x$1" : 'x-J' >/dev/null; do
    opt=`expr "$1" : '-J\(.*\)'`
    javaOpts="${javaOpts} -${opt}"
    shift
done

exec java ${javaOpts} -jar ${jardir}/${jarfile} "$@"
