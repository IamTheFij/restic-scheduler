#! /bin/bash
set -ex

# Create flat file
echo "Hello" > /data/test.txt

# Create Sqlite database
touch /data/test_database.db
sqlite3 /data/test_database.db <<-EOF
CREATE TABLE test_table (
  id integer PRIMARY KEY,
  data text NOT NULL,
);

INSERT INTO test_table(data)
VALUES ("Test row");
EOF

# Create MySql database
mysql --user "$MYSQL_USER" --password "$MYSQL_PWD" main <<-EOF
CREATE TABLE test_table (
  id integer AUTO_INCREMENT PRIMARY KEY,
  data text NOT NULL,
);

INSERT INTO test_table(data)
VALUES ("Test row");
EOF

# Create Postgresql database
pgsql --username "$PGSQL_USER" --dbname main <<-EOF
CREATE TABLE test_table (
  id integer PRIMARY KEY,
  data text NOT NULL,
);

INSERT INTO test_table(data)
VALUES ("Test row");
EOF
