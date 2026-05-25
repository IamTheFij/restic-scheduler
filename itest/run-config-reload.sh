#! /bin/bash
set -ex

cd "$(dirname "$0")"
mkdir -p ./repo ./data

echo "Clean everything"
docker compose down -v --remove-orphans
rm -fr ./repo/* ./data/*
sleep 5

echo "Boostrap databases and data"
docker compose up -d mariadb postgres
docker compose run --rm bootstrap
sleep 1

echo "Back up job file and create dummy job"
cp test-backup.hcl test-backup.hcl.bak
cat <<EOH > ./test-backup.hcl
job "Dummy" {
  schedule = "* * * * *"

  config {
    repo = "/repo"
    passphrase = "shh"
  }

  backup {
    paths = ["/dev/null"]
  }
}
EOH

echo "Start backup job"
docker compose up -d main

# Run for at least 1 minute
timeout --signal=SIGINT 70s docker compose logs -f main || echo ok

echo "Make sure only dummy backup task ran"
docker compose logs main | grep -q "JobName:Dummy JobType:backup Success:true"
docker compose logs main | grep -vq "IntegrationTest"

echo "Replace backup file"
cat test-backup.hcl.bak > test-backup.hcl

echo "Send SIGHUP to reload"
docker compose kill --signal=SIGHUP main

# Run for at least 1 minute
timeout --signal=SIGINT 70s docker compose logs --tail 10 -f main || echo ok

echo "Make sure new backup task ran"
docker compose logs main | grep -q "Configuration reload successful"
docker compose logs main | grep -q "JobName:IntegrationTest JobType:backup Success:true"

# Check container health
docker compose ps | grep -q "(healthy)"

echo "Stop and clean data"
rm -fr ./data/*
docker compose down -v
docker compose up -d mariadb postgres
docker compose run --rm db-wait
sleep 1

echo "Run restore"
docker compose run --rm main -restore IntegrationTest -once /test-backup.hcl
sleep 1

echo "Validate data"
docker compose run --rm validate

echo "Clean all again"
docker compose down -v --remove-orphans
rm -fr ./repo/* ./data/*
