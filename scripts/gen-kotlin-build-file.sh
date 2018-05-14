#!/data/data/com.termux/files/usr/bin/bash
set -e

if [ -z "$1" ]; then
  echo "usage: $0 <classpath> <outDir> <rspFiles>..." >&2
  exit 1
fi

if [[ $1 == "-classpath" ]]; then
  shift
fi;

classpath=$1
out_dir=$2
shift 2

prefix=`pwd`

echo "<modules><module name=\"name\" type=\"java-production\" outputDir=\"${out_dir}\">"

for file in $(echo $classpath | tr ":" "\n"); do
  echo "  <classpath path=\"${prefix}/${file}\"/>"
done

while (( "$#" )); do
  for file in $(cat $1); do
    if [[ $file == *.java ]]; then
      echo "  <javaSourceRoots path=\"${prefix}/${file}\"/>"
    elif [[ $file == *.kt ]]; then
      echo "  <sources path=\"${prefix}/${file}\"/>"
    else
      echo "Unknown source file type ${file}"
      exit 1
    fi
  done

  shift
done

echo "</module></modules>"
