#!/bin/bash

cd ../extension
echo "Packing language syntax"
vsce package
cd ../scripts
