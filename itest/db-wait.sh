#! /bin/sh
set -ex

echo "Wait for mariadb"
until mariadb --skip-ssl --host "$MYSQL_HOST" --user "$MYSQL_USER" --password="$MYSQL_PWD" --execute "SHOW DATABASES;"; do
  sleep 1
done

echo "Wait for postgres"
# Create Postgres database
export PGPASSWORD="$PGSQL_PASS"
until psql --host "$PGSQL_HOST" --username "$PGSQL_USER" --command "SELECT datname FROM pg_database;"; do
  sleep 1
done
