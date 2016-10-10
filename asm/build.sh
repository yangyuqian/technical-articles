#!/bin/sh

function help(){
echo 'Usage: sh build.sh <example>

Example:
$ sh build.sh e1
  Compile examples/e1.asm and put the executable at paly/e1'
}

if [ -z "$1" ]; then
  help
  exit 1
fi

CUR_DIR=${PWD}
BASEDIR=$(dirname "$0")

if [ "$BASEDIR" != "." ]; then
  echo "build.sh must be executed under asm ..."
  exit 1
fi

asm="$1"
# remove the .asm suffix
asm=`echo $asm|sed -e "s/\.asm//g"`

# exit if file missing
if [ ! -f "$CUR_DIR/examples/${asm}.asm" ]; then
  echo "Example $asm not found ..."
  exit 1
fi

# generate object files
obj=$CUR_DIR/play/${asm}.o
fmt=macho64
asmpath=${CUR_DIR}/examples/${asm}.asm
exe=${CUR_DIR}/play/$asm
entry='_main'

nasm -f $fmt -o $obj $asmpath
ld -o $exe -e $entry $obj

case "$2" in
exec)
echo "\n\n=============== Run example $asm ==============\n\n"
set -x
$exe
;;
*)
  echo 'Executable of '${asm}' is located at '$exe
esac
