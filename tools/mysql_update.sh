#!/bin/sh

# Update the mysql database with the new lines schedules.
# Arg 1 is the path to the file holding database password
# Arg 2 is the user@host SSH connection string

MYSQL=mysql
MYSQLDB=businfo
SQLDB=/tmp/htdb.sql
CMD=/tmp/cmd

error() {
    echo "Error: $1"
    exit 1
}

log() {
    echo "===> $1 ..."
}

if [ -z "$1" ]; then
    error "missing user@host or local as arg 1"
fi

if [ -z "$2" ]; then
    error "missing path to database password file"
fi

TARGET=$1
PASSWD=$(cat $2)
MYSQLOPTS="-u businfo --password=$PASSWD"

log "Upgrading $TARGET MySQL database"
log "Deleting old tables and pushing new content"

if [ $TARGET = "local" ]; then
    test -r "$2" || error "can't access $2 for reading"
    for TABLE in stop line_station station line city network; do
        ${MYSQL} ${MYSQLOPTS} ${MYSQLDB} -e "DROP TABLE IF EXISTS $TABLE"
    done
    ${MYSQL} ${MYSQLOPTS} ${MYSQLDB} < ${SQLDB}
else
    log "sending SQL update file"
    scp $SQLDB $TARGET:/tmp
    log "executing update script"
    cat > $CMD << EOF
PFILE="$2"
if [ ! -r \$PFILE ]; then
    echo "error: can't access \$PFILE for reading"
    exit 1
fi
PWD=\$(cat \$PFILE)
EOF
    for TABLE in stop line_station station line city network; do
        echo "${MYSQL} -u businfo --password=\$PWD ${MYSQLDB} -e \"DROP TABLE IF EXISTS $TABLE\"" >> $CMD
    done
    echo "${MYSQL} -u businfo --password=\$PWD ${MYSQLDB} < ${SQLDB}" >> $CMD
    ssh $TARGET "`cat $CMD`"
    rm $CMD
fi
