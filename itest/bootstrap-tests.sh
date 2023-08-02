#! /bin/sh
set -ex

# Create flat file
echo "Hello" > /data/test.txt

# Create Sqlite database
touch /data/test_database.db
sqlite3 /data/test_database.db <<-EOF
CREATE TABLE test_table (
  id integer PRIMARY KEY,
  data text NOT NULL
);

INSERT INTO test_table(data)
VALUES ("Test row");
EOF

# Create MySql database
until mysql --host "$MYSQL_HOST" --user "$MYSQL_USER" --password="$MYSQL_PWD" --execute "SHOW DATABASES;"; do
  sleep 1
done
mysql --host "$MYSQL_HOST" --user "$MYSQL_USER" --password="$MYSQL_PWD" main <<EOF
CREATE TABLE test_table (
  id integer AUTO_INCREMENT PRIMARY KEY,
  data text NOT NULL
);

INSERT INTO test_table(data)
VALUES ("Test row");
EOF
