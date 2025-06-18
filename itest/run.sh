#! /bin/bash
set -ex

cd "$(dirname "$0")"
mkdir -p ./repo ./data

echo Clean everything
docker-compose down -v --remove-orphans
rm -fr ./repo/* ./data/*
sleep 5

echo Boostrap databases and data
docker-compose up -d mariadb postgres
docker-compose run --rm bootstrap
sleep 1

echo Run backup job
docker-compose run --rm main -backup IntegrationTest -once /test-backup.hcl

echo Clean data
docker-compose down -v
docker-compose up -d mariadb postgres
rm -fr ./data/*
sleep 15

echo Run restore
docker-compose run --rm main -restore IntegrationTest -once /test-backup.hcl
sleep 1

echo Validate data
docker-compose run --rm validate

echo Clean all again
docker-compose down -v --remove-orphans
rm -fr ./repo/* ./data/*
