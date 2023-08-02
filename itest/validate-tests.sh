#! /bin/sh
set -ex

# Check flat file
test -f /data/test.txt
grep "^Hello" /data/test.txt

# Check Sqlite database
test -f /data/test_database.db
sqlite3 /data/test_database.db "select data from test_table where id = 1" | grep "^Test row"

# Check MySql database
mysql --host "$MYSQL_HOST" --user "$MYSQL_USER" --password="$MYSQL_PWD" main <<-EOF | grep "^Test row"
select data from test_table where id = 1;
EOF
