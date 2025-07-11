#!/bin/bash

cd ../compiler/cmd
go build -o ../bin/ferret -ldflags "-s -w" -trimpath -v
