#!/bin/bash

cd ../compiler
go build -o ../bin/ferret -ldflags "-s -w" -trimpath -v
