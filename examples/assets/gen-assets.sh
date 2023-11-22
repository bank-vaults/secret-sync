#!/usr/bin/env bash

# Requirement: https://github.com/terrastruct/d2

ASSET_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

for d2_file in "$ASSET_DIR"/*.d2
do
  # Get files
  png_file=${d2_file%".d2"}.png

  # Generate PNG from diagram
  d2 $d2_file $png_file
done
