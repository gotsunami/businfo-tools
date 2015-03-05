
SQLDB=/tmp/businfo/htdb.sql
DBOPTS=--gps --gps-cache=gps.csv --pre-filter=filter.map
MAKERES=./tools/makeres.py
SQLITEDB=~/ht.sqlite
SQLITE=sqlite3
CHKSUM=/tmp/businfo/.checksum
LINECOMPILER=bsc
LINECOMPILERSRCPATH=bin/${LINECOMPILER}
LINECOMPILERSRC=${LINECOMPILERSRCPATH}/${LINECOMPILER}.go

.PHONY: makedb clean test sqlite mysql mysqldb

all: sqlite

makedb: bsc
	@echo "Generating raw SQLite content..."
	@${MAKERES} ${DBOPTS} sqlite raw/

mysqldb: bsc
	@echo "Generating raw MySQL content..."
	@${MAKERES} ${DBOPTS} mysql raw/

bsc:
	@go build -o ${LINECOMPILERSRCPATH}/${LINECOMPILER} ${LINECOMPILERSRC}

sqlite: makedb
	@echo "Making SQLite database..."
	@rm -f ${SQLITEDB} && ${SQLITE} ${SQLITEDB} < ${SQLDB}
	@echo "Wrote in ${SQLITEDB}"

deploy-local: mysqldb
	@./tools/mysql_update.sh local .bpw

clean:
	rm -rf /tmp/businfo && rm -f ${LINECOMPILERSRCPATH}/${LINECOMPILER}
