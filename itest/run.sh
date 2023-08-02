#! /bin/bash
set -ex

echo Clean everything
docker-compose down -v
rm -fr ./repo/* ./data/*

echo Boostrap databases and data
docker-compose up -d mysql
docker-compose run bootstrap /bootstrap-tests.sh

echo Run backup job
docker-compose run main -backup IntegrationTest -once /test-backup.hcl

echo Clean data
docker-compose down -v
rm -fr ./data/*

echo Run restore
docker-compose run main -restore IntegrationTest -once /test-backup.hcl

echo Validate data
docker-compose run validate /validate-tests.sh

echo Clean all again
docker-compose down -v
rm -fr ./repo/* ./data/*
