#!/bin/bash

EXAMPLE=$1

if [[ "" == "$EXAMPLE" ]]; then
  echo "Pick the example folder"
  exit 1
fi

cd $EXAMPLE
bash run.sh