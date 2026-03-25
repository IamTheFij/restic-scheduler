#! /bin/bash
set -ex

cd "$(dirname "$0")"
mkdir -p ./repo ./data

echo "Clean everything"
docker compose down -v --remove-orphans
rm -fr ./repo/* ./data/*
