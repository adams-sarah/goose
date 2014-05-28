package goose

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"
)

type MigrationExecutor struct {
	Conf           *DBConf
	CurrentVersion int64
	DB             *sql.DB
}

func NewMigrationExecutor() *MigrationExecutor {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	path := os.Getenv("GOOSE_DB_PATH")
	if path == "" {
		path = "db"
	}

	conf, err := NewDBConf(path, env)
	if err != nil {
		log.Fatal(err)
	}

	db, e := sql.Open(conf.Driver.Name, conf.Driver.OpenStr)
	if e != nil {
		log.Fatal("couldn't open DB:", e)
	}

	currentVersion, err := EnsureDBVersion(conf, db)
	if err != nil {
		log.Fatal(err)
	}

	return &MigrationExecutor{conf, currentVersion, db}
}

func (me *MigrationExecutor) Do(methodName string, args ...interface{}) {
	var zeroValue reflect.Value

	methodName = strings.Title(methodName)

	method := reflect.ValueOf(me).MethodByName(methodName)

	if method == zeroValue {
		log.Fatalf("MigrationExecutor.Do: no method found with name %s", methodName)
	}

	methodArgs := []reflect.Value{}

	for _, arg := range args {
		methodArgs = append(methodArgs, reflect.ValueOf(arg))
	}

	method.Call(methodArgs)
}

func (me *MigrationExecutor) Up(runCount ...int64) {
	if len(runCount) > 0 {
		me.run(runCount[0], UP)
	} else {
		me.run(RUN_ALL, UP)
	}

}

func (me *MigrationExecutor) Down(runCount ...int64) {
	if len(runCount) > 0 {
		me.run(runCount[0], DOWN)
	} else {
		me.run(1, DOWN)
	}
}

func (me *MigrationExecutor) Redo(runCount ...int64) {
	if len(runCount) > 0 {
		me.run(runCount[0], DOWN, UP)
	} else {
		me.run(1, DOWN, UP)
	}
}

func (me *MigrationExecutor) Status() {
	defer me.DB.Close()

	me.printMigrationStatuses()
}

func (me *MigrationExecutor) Create(migrationName string) {
	absolutePath := CreateGoMigration(migrationName, me.Conf.MigrationsDir)

	fmt.Println("goose: created", absolutePath)
}

func (me *MigrationExecutor) Force(migrationVersion int64) {
	defer me.DB.Close()

	if UserMigrations[migrationVersion].Version == int64(0) {
		fmt.Println("goose: no migration found for version", migrationVersion)
	} else {
		forceMigrationVersion(me.DB, me.Conf, migrationVersion)
		log.Printf("OK   %d_%s\n", migrationVersion, UserMigrations[migrationVersion].Name)
	}
}

func (me *MigrationExecutor) run(runCount int64, directions ...bool) {
	defer me.DB.Close()
	for _, direction := range directions {
		if err := RunMigrations(me, direction, runCount); err != nil {
			log.Fatal(err)
		}
	}
}

func (me *MigrationExecutor) printMigrationStatuses() {
	fmt.Printf("goose: status for environment '%v'\n", me.Conf.Env)
	fmt.Println("    Applied At                      Migration")
	fmt.Println("    ================================================================================")

	sortedVersions := []int64{}
	for v, _ := range UserMigrations {
		sortedVersions = append(sortedVersions, v)
	}
	sort.Sort(int64arr(sortedVersions))

	mAppliedLookup := appliedMigrationsLookup(me.DB)

	for _, version := range sortedVersions {
		var appliedAt string

		if !mAppliedLookup[version].IsZero() {
			appliedAt = mAppliedLookup[version].Format(time.ANSIC)
		} else {
			appliedAt = "Pending"
		}

		fmt.Printf("    %-24s   --   %d_%s\n", appliedAt, version, UserMigrations[version].Name)
	}

}
