#!/bin/bash

clear

echo Building Ferret...

cd ../compiler/cmd
go build -o ../../bin/ferret -ldflags "-s -w" -trimpath -v
if [ $? -ne 0 ]; then
    echo "Build failed. Exiting..."
    exit 1
fi
echo "Running Ferret..."
cd ../../bin
./ferret "./../app/cmd/start.fer" -debug
if [ $? -ne 0 ]; then
    echo "Ferret execution failed. Exiting..."
    exit 1
fi
echo "Ferret build and run completed successfully."