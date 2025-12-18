#! /bin/sh
set -ex

# Create flat file
echo "Hello" > /data/test.txt

# Create Sqlite database
touch /data/test_database.db
sqlite3 /data/test_database.db <<-EOF
CREATE TABLE test_table (
  id INTEGER PRIMARY KEY,
  data TEXT NOT NULL
);

INSERT INTO test_table(data)
VALUES ("Test row");
EOF

# Create MySql database
until mariadb --skip-ssl --host "$MYSQL_HOST" --user "$MYSQL_USER" --password="$MYSQL_PWD" --execute "SHOW DATABASES;"; do
  sleep 1
done
mariadb --skip-ssl --host "$MYSQL_HOST" --user "$MYSQL_USER" --password="$MYSQL_PWD" main <<EOF
CREATE TABLE test_table (
  id INTEGER AUTO_INCREMENT PRIMARY KEY,
  data TEXT NOT NULL
);

INSERT INTO test_table(data)
VALUES ("Test row");
EOF

# Create Postgres database
export PGPASSWORD="$PGSQL_PASS"
until psql --host "$PGSQL_HOST" --username "$PGSQL_USER" --command "SELECT datname FROM pg_database;"; do
  sleep 1
done
psql -v ON_ERROR_STOP=1 --host "$PGSQL_HOST" --username "$PGSQL_USER" main <<EOF
CREATE TABLE test_table (
  id SERIAL PRIMARY KEY,
  data TEXT NOT NULL
);

INSERT INTO test_table(data)
VALUES ('Test row');
EOF
