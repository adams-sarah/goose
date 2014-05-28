package goose

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	_ "github.com/ziutek/mymysql/godrv"
)

func runGoMigration(db *sql.DB, conf *DBConf, version int64, direction bool) error {
	txn, err := db.Begin()
	if err != nil {
		log.Fatal("db.Begin:", err)
	}

	directionFn := UserMigrations[version].Down
	if direction {
		directionFn = UserMigrations[version].Up
	}

	err = directionFn(txn)
	if err != nil {
		txn.Rollback()
		return err
	}

	stmt := conf.Driver.Dialect.insertVersionSql()
	if _, err = txn.Exec(stmt, version, direction); err != nil {
		txn.Rollback()
		return err
	}

	txn.Commit()

	return nil
}

func forceMigrationVersion(db *sql.DB, conf *DBConf, version int64) error {
	stmt := conf.Driver.Dialect.insertVersionSql()
	_, err := db.Exec(stmt, version, UP)
	return err
}
