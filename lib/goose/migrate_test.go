package goose

import (
	"database/sql"
	"testing"
)

var myPackageVar string

// Register migrations for goose to run
func TestRegisterMigration(t *testing.T) {
	RegisterMigration(int64(20140406181655), "AddTestFieldToTable", Up_20140406181655, Down_20140406181655)

	if len(UserMigrations) < 1 {
		t.Errorf("Migration did not register.")
	}

	if UserMigrations[int64(20140406181655)].Version != int64(20140406181655) {
		t.Errorf("Migration registered with incorrect version. Expected %d got %d", 20140406181655, UserMigrations[int64(20140406181655)].Version)
	}
}

func TestRunMigrations(t *testing.T) {
	me := NewMigrationExecutor()
	err := RunMigrations(me, UP, RUN_ALL)
	if err != nil {
		t.Errorf("Migration returned error: %s.", err.Error())
	}

	expected := "Up"
	if myPackageVar != expected {
		t.Errorf("Migration did not run UP. Expected myPackageVar to equal %s, got %s.", expected, myPackageVar)
	}

	err = RunMigrations(me, DOWN, RUN_ALL)
	if err != nil {
		t.Errorf("Migration returned error: %s.", err.Error())
	}

	expected = "Down"
	if myPackageVar != expected {
		t.Errorf("Migration did not run DOWN. Expected myPackageVar to equal %s, got %s.", expected, myPackageVar)
	}
}

// Up is executed when this migration is applied
func Up_20140406181655(txn *sql.Tx) (err error) {
	myPackageVar = "Up"
	return
}

// Down is executed when this migration is rolled back
func Down_20140406181655(txn *sql.Tx) (err error) {
	myPackageVar = "Down"
	return
}
