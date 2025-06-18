#! /bin/sh
set -ex

# Check flat file
test -f /data/test.txt
grep "^Hello" /data/test.txt

# Pre-backup should be found
test -f /data/pre-backup.txt
grep "^HelloPreBackup" /data/pre-backup.txt
# Post-backup should not be found
test ! -f /data/post-backup.txt

# Pre-restore file that doesn't collide should be found
test -f /data/pre-restore.txt
grep "^HelloPreRestore" /data/pre-restore.txt

# on-restore should be found and pre-restore value should be gone
test -f /data/on-restore.txt
grep -v "^Pre" /data/on-restore.txt
grep "^Post" /data/on-restore.txt

# Check Sqlite database
test -f /data/test_database.db
sqlite3 /data/test_database.db "select data from test_table where id = 1" | grep "^Test row"

# Check MySql database
until mysql --host "$MYSQL_HOST" --user "$MYSQL_USER" --password="$MYSQL_PWD" --execute "SHOW DATABASES;"; do
  sleep 1
done
mysql --host "$MYSQL_HOST" --user "$MYSQL_USER" --password="$MYSQL_PWD" main <<EOF | grep "^Test row"
select data from test_table where id = 1;
EOF

# Check Postgres database
export PGPASSWORD="$PGSQL_PASS"
until psql --host "$PGSQL_HOST" --username "$PGSQL_USER" --command "SELECT datname FROM pg_database;"; do
  sleep 1
done
psql --host "$PGSQL_HOST" --user "$PGSQL_USER" main <<EOF | grep "Test row"
select data from test_table where id = 1;
EOF
