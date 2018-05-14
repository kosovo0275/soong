#!/data/data/com.termux/files/usr/bin/bash
set -eu

function die() {
  echo "$@" >&2
  exit 1
}

function usage() {
  violation="$1"
  die "$violation

  Usage: diff_build_graphs.sh [--products=product1,product2...] <OLD_VERSIONS> <NEW_VERSIONS>

  This file builds and parses the build files (Android.mk, Android.bp, etc) for each requested
  product and for both sets of versions, and checks whether the ninja files (which implement
  the build graph) changed between the two versions.

  Example: diff_build_graphs.sh 'build/soong:work^ build/blueprint:work^' 'build/soong:work build/blueprint:work'

  Options:
    --products=PRODUCTS  comma-separated list of products to check"
}

PRODUCTS_ARG=""
OLD_VERSIONS=""
NEW_VERSIONS=""
function parse_args() {
  while true; do
    arg="${1-}"
    case "$arg" in
      --products=*) PRODUCTS_ARG="$arg";;
      *) break;;
    esac
    shift
  done
  # parse required arguments
  if [ "$#" != "2" ]; then
    usage ""
  fi
  #argument validation
  OLD_VERSIONS="$1"
  NEW_VERSIONS="$2"

}
parse_args "$@"


builtin cd "$(dirname $0)"
SCRIPT_DIR="$PWD"
builtin cd ../../..
CHECKOUT_ROOT="$PWD"
OUT_DIR="${OUT_DIR-}"
if [ -z "$OUT_DIR" ]; then
  OUT_DIR=out
fi
WORK_DIR="$OUT_DIR/diff"
OUT_DIR_OLD="$WORK_DIR/out_old"
OUT_DIR_NEW="$WORK_DIR/out_new"
OUT_DIR_TEMP="$WORK_DIR/out_temp"


function checkout() {
  versionSpecs="$1"
  for versionSpec in $versionSpecs; do
    project="$(echo $versionSpec | sed 's|\([^:]*\):\([^:]*\)|\1|')"
    ref="$(echo     $versionSpec | sed 's|\([^:]*\):\([^:]*\)|\2|')"
    echo "checking out ref $ref in project $project"
    git -C "$project" checkout "$ref"
  done
}

function run_build() {
  echo
  echo "Starting build"
  make blueprint_tools
  builder="$(echo $OUT_DIR/soong/host/*/bin/multiproduct_kati)"
  BUILD_NUMBER=sample "$builder" $PRODUCTS_ARG --keep --out "$OUT_DIR_TEMP" || true
  echo
}

function diffProduct() {
  product="$1"

  zip1="$OUT_DIR_OLD/${product}.zip"
  unzipped1="$OUT_DIR_OLD/$product"

  zip2="$OUT_DIR_NEW/${product}.zip"
  unzipped2="$OUT_DIR_NEW/$product"

  unzip -qq "$zip1" -d "$unzipped1"
  unzip -qq "$zip2" -d "$unzipped2"

  diffFile="$WORK_DIR/diff.txt"
  diff -r "$unzipped1" "$unzipped2" -x build_date.txt -x build_number.txt -x '\.*' -x '*.log' -x build_fingerprint.txt -x build.ninja.d -x '*.zip' > $diffFile || true
  if [[ -s "$diffFile" ]]; then
    echo "First few differences (total diff linecount=$(wc -l $diffFile)) for product $product:"
    cat "$diffFile" | head -n 10
    echo "End of differences for product $product"
    rm -rf "$unzipped1" "$unzipped2"
  else
    rm -rf "$zip1" "$unzipped1" "$zip2" "$unzipped2"
  fi
}

function do_builds() {
  rm -rf "$WORK_DIR"
  mkdir "$WORK_DIR"

  checkout "$NEW_VERSIONS"
  run_build
  mv "$OUT_DIR_TEMP" "$OUT_DIR_NEW"

  checkout "$OLD_VERSIONS"
  run_build
  mv "$OUT_DIR_TEMP" "$OUT_DIR_OLD"

  echo created "$OUT_DIR_OLD" and "$OUT_DIR_NEW"
}

function main() {
  do_builds
  checkout "$NEW_VERSIONS"

  productsFile="$WORK_DIR/all_products.txt"
  find $OUT_DIR_OLD $OUT_DIR_NEW -mindepth 1 -maxdepth 1 -name "*.zip" | sed "s|^$OUT_DIR_OLD/||" | sed "s|^$OUT_DIR_NEW/||" | sed "s|\.zip$||" | sort | uniq > "$productsFile"
  echo Diffing products
  for product in $(cat $productsFile); do
    diffProduct "$product"
  done
  echo Done diffing products
  echo "Any differing outputs can be seen at $OUT_DIR_OLD/*.zip and $OUT_DIR_NEW/*.zip"
  echo "See $WORK_DIR/diff.txt for the full list of differences for the latest product checked"
}

main
