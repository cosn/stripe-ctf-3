#!/bin/sh

set -e

# Add or modify any build steps you need here

cd "$(dirname "$0")"
cp src/words words
go build -o ics src/ics/*.go 
