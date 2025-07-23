#!/bin/bash

clear
cd ../compiler/cmd
go run . "./../../app/test_builtin_import.fer" -debug
