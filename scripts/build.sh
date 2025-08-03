#!/bin/bash

cd ..
go build -o bin/ferret -ldflags "-s -w" -trimpath -v ./compiler
