#!/data/data/com.termux/files/usr/bin/bash
set -e

if [ -z "$1" ]; then
  echo "usage: $0 <module name> <jar1> [<jar2> ...]" >&2
  exit 1
fi

module_name=$1
shift

echo "module ${module_name} {"
for j in "$@"; do zipinfo -1 $j ; done \
  | grep -E '/[^/]*\.class$' \
  | sed 's|\(.*\)/[^/]*\.class$|    exports \1;|g' \
  | sed 's|/|.|g' \
  | sort -u
echo "}"
