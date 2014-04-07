package goose

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/ziutek/mymysql/godrv"
	"log"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"
)

const VERSION_LAYOUT = "20060102150405"

// List of migrations we can run for the user, ie.
// map[version]Migration
var UserMigrations = map[int64]Migration{}

type int64arr []int64

func (a int64arr) Len() int           { return len(a) }
func (a int64arr) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a int64arr) Less(i, j int) bool { return a[i] < a[j] }

var (
	ErrTableDoesNotExist = errors.New("table does not exist")
	ErrNoPreviousVersion = errors.New("no previous version found")
)

type MigrationRecord struct {
	VersionId int64
	TStamp    time.Time
	IsApplied bool // was this a result of up() or down()
}

type Migration struct {
	Version int64
	Name    string
	Up      func(*sql.Tx) error
	Down    func(*sql.Tx) error
}

func RegisterMigration(v int64, migrationName string, upfn func(*sql.Tx) error, downfn func(*sql.Tx) error) {
	if UserMigrations[v].Version > 0 {
		log.Fatalf("More than one migration specified for version %d", v)
	}

	UserMigrations[v] = Migration{
		Version: v,
		Name:    migrationName,
		Up:      upfn,
		Down:    downfn,
	}
}

func RunMigrations(me *MigrationExecutor, isUp bool, runCount int64) (err error) {
	migrations, err := CollectMigrations(me.DB, isUp, runCount)
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Printf("goose: no migrations to run. current version: %d\n", me.CurrentVersion)
		return nil
	}

	directionStr := "DOWN"
	if isUp {
		directionStr = "UP"
	}

	fmt.Printf("goose: migrating db environment '%v', current version: %d, direction: %s\n",
		me.Conf.Env, me.CurrentVersion, directionStr)

	for _, m := range migrations {

		err = runGoMigration(me.DB, me.Conf, m.Version, isUp)

		if err != nil {
			log.Printf("FAIL   %d_%s     %s\n", m.Version, m.Name, err.Error())
		} else {
			log.Printf("OK     %d_%s\n", m.Version, m.Name)
		}
	}

	return nil
}

// collect all the valid looking migration scripts in the
// migrations folder, and key them by version
func CollectMigrations(db *sql.DB, isUp bool, runCount int64) (m []Migration, err error) {

	if len(UserMigrations) == 0 {
		return m, nil
	}

	sortedVersions := []int64{}
	for v, _ := range UserMigrations {
		sortedVersions = append(sortedVersions, v)
	}
	sort.Sort(int64arr(sortedVersions))

	mAppliedLookup := appliedMigrationsLookup(db)

	for _, v := range sortedVersions {
		if !mAppliedLookup[v].IsZero() && !isUp {
			m = append(m, UserMigrations[v])
		}

		if mAppliedLookup[v].IsZero() && isUp {
			m = append(m, UserMigrations[v])
		}
	}

	if runCount > 0 {
		m = m[(int64(len(m)) - runCount):]
	}

	if !isUp {
		// reverse order of migrations
		for i, j := 0, len(m)-1; i < j; i, j = i+1, j-1 {
			m[i], m[j] = m[j], m[i]
		}
	}

	return m, nil
}

func appliedMigrationsLookup(db *sql.DB) map[int64]time.Time {
	q := fmt.Sprintf("SELECT tstamp, version_id, is_applied FROM goose_db_version WHERE id IN (SELECT MAX(id) AS id FROM goose_db_version GROUP BY version_id);")
	rows, e := db.Query(q)

	if e != nil {
		log.Fatal(e.Error())
	}

	defer rows.Close()

	versionIds := map[int64]time.Time{}
	for rows.Next() {
		var row MigrationRecord
		rows.Scan(&row.TStamp, &row.VersionId, &row.IsApplied)
		if row.IsApplied {
			versionIds[row.VersionId] = row.TStamp
		}
	}

	return versionIds
}

// retrieve the current version for this DB.
// Create and initialize the DB version table if it doesn't exist.
func EnsureDBVersion(conf *DBConf, db *sql.DB) (int64, error) {

	rows, err := conf.Driver.Dialect.dbVersionQuery(db)
	if err != nil {
		if err == ErrTableDoesNotExist {
			return 0, createVersionTable(conf, db)
		}
		return 0, err
	}
	defer rows.Close()

	// The most recent record for each migration specifies
	// whether it has been applied or rolled back.
	// The first version we find that has been applied is the current version.

	toSkip := make([]int64, 0)

	for rows.Next() {
		var row MigrationRecord
		if err = rows.Scan(&row.VersionId, &row.IsApplied); err != nil {
			log.Fatal("error scanning rows:", err)
		}

		// have we already marked this version to be skipped?
		skip := false
		for _, v := range toSkip {
			if v == row.VersionId {
				skip = true
				break
			}
		}

		// if version has been applied and not marked to be skipped, we're done
		if row.IsApplied && !skip {
			return row.VersionId, nil
		}

		// version is either not applied, or we've already seen a more
		// recent version of it that was not applied.
		if !skip {
			toSkip = append(toSkip, row.VersionId)
		}
	}

	panic("failure in EnsureDBVersion()")
}

// Create the goose_db_version table
// and insert the initial 0 value into it
func createVersionTable(conf *DBConf, db *sql.DB) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	d := conf.Driver.Dialect

	if _, err := txn.Exec(d.createVersionTableSql()); err != nil {
		txn.Rollback()
		return err
	}

	version := 0
	applied := true
	if _, err := txn.Exec(d.insertVersionSql(), version, applied); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}

func CreateGoMigration(name, dir string) string {
	if err := os.MkdirAll(dir, 0777); err != nil {
		log.Fatal(err)
	}

	t := time.Now()
	timestamp := t.Format(VERSION_LAYOUT)
	filename := fmt.Sprintf("%v_%v.go", timestamp, name)

	fpath := filepath.Join(dir, filename)

	tmpl := goMigrationTemplate
	params := map[string]interface{}{
		"Version": timestamp,
		"Name":    name,
	}

	path, err := writeTemplateToFile(fpath, tmpl, params)
	if err != nil {
		log.Fatal(err)
	}

	a, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	return a
}

var goMigrationTemplate = template.Must(template.New("goose.go-migration").Parse(`
package migrations

import (
	"database/sql"
	"github.com/Sproutling/goose/lib/goose"
)

// Register migrations for goose to run
func init() {
	goose.RegisterMigration(int64({{ .Version }}), "{{.Name}}", Up_{{ .Version }}, Down_{{ .Version }})
}

// Up is executed when this migration is applied
func Up_{{ .Version }}(txn *sql.Tx) (err error) {
	_, err = txn.Exec("")
	return err
}

// Down is executed when this migration is rolled back
func Down_{{ .Version }}(txn *sql.Tx) (err error) {
	_, err = txn.Exec("")
	return err
}
`))
