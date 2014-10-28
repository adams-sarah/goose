# goose

This is a fork of the [original goose](https://bitbucket.org/liamstask/goose) by liamstask. This fork has removed the need for the go executable at runtime, which is helpful for hosted environments. We also removed support for straight SQL migrations, for lack of time and resources.

goose is a database migration tool.

You can manage your database's evolution by creating incremental Go migration scripts.

# Installation

    $ go get bitbucket.org/Sproutling/goose/lib/goose

# Usage

goose provides several methods to help manage your database schema.

You can access these methods easily using go's built-in flag support:

Example Usage:

```go
var dbmigratecreate = flag.String("db:migrate:create", "", "create a new migration with the given name, ie. AddTestFieldToTable")
var dbmigratedown = flag.Int64("db:migrate:down", 0, "migrate db down N migrations; -1 signifies all migrations")
var dbmigrateup = flag.Int64("db:migrate:up", 0, "migrate db up N migrations; -1 signifies all migrations")
var dbmigrateredo = flag.Int64("db:migrate:redo", 0, "migrate db down N migrations, then back up N migrations; -1 signifies all migrations")
var dbmigrateforceall = flag.Bool("db:migrate:force:all", false, "mark all migrations as 'ran' - useful for setting up a new dev environment")
var dbmigratestatus = flag.Bool("db:migrate:status", false, "print db migration status")

func main() {
    flag.Parse()

	switch {
	case *dbmigratecreate != "":
		runDBMigrateCreate(*dbmigratecreate)
		return
	case *dbmigratedown != int64(0):
		runDBMigrate("Down", *dbmigratedown)
		return
	case *dbmigrateup != int64(0):
		runDBMigrate("Up", *dbmigrateup)
		return
	case *dbmigrateredo != int64(0):
		runDBMigrate("Redo", *dbmigrateredo)
		return
	case *dbmigrateforceall:
		runDBMigrateForceAll()
		return
	case *dbmigratestatus:
		runDBMigrateStatus()
		return
	}
}

func runDBMigrate(cmd string, count int64) {
	migrationExec := goose.NewMigrationExecutor()
	migrationExec.Do(cmd, count)
}

func runDBMigrateCreate(migrationName string) {
	migrationExec := goose.NewMigrationExecutor()
	migrationExec.Do("Create", migrationName)
}

func runDBMigrateStatus() {
	migrationExec := goose.NewMigrationExecutor()
	migrationExec.Do("Status")
}

func runDBMigrateForceAll() {
	migrationExec := goose.NewMigrationExecutor()
	migrationExec.Do("ForceAll")
}
```

```bash
$ myapp -db:migrate:up -1
```


# Configuration

goose expects you to maintain a folder (typically called "db"), which contains the following:

* a dbconf.yml file that describes the database configurations you'd like to use
* a folder called "migrations" which contains .go scripts that implement your migrations

A sample dbconf.yml looks like

    development:
        driver: postgres
        open: user=liam dbname=tester sslmode=disable

Here, `development` specifies the name of the environment, and the `driver` and `open` elements are passed directly to database/sql to access the specified database.
