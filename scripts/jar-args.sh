#!/data/data/com.termux/files/usr/bin/bash
set -e

if [ "$1" == "--test" ]; then
  in=$(mktemp)
  expected=$(mktemp)
  out=$(mktemp)
  cat > $in <<EOF
a
a/b
a/b/'
a/b/"
a/b/\\
a/b/#
a/b/a
EOF
  cat > $expected <<EOF

-C 'a' 'b'
-C 'a' 'b/\\''
-C 'a' 'b/"'
-C 'a' 'b/\\\\'
-C 'a' 'b/#'
-C 'a' 'b/a'
EOF
  cat $in | $0 a > $out

  if cmp $out $expected; then
    status=0
    echo "PASS"
  else
    status=1
    echo "FAIL"
    echo "got:"
    cat $out
    echo "expected:"
    cat $expected
  fi
  rm -f $in $expected $out
  exit $status
fi

sed -r \
  -e"s,^$1(/|\$),," \
  -e"s,(['\\]),\\\\\1,g" \
  -e"s,^(.+),-C '$1' '\1',"
