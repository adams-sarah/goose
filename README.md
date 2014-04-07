# goose

This is a fork of the [original goose](https://bitbucket.org/liamstask/goose) by liamstask. This fork has removed the need for a go binary in `$PATH`, which is helpful for hosted environments (ie. Heroku). We also removed support for straight SQL migrations for lack of time and resources.

goose is a database migration tool.

You can manage your database's evolution by creating incremental Go migration scripts.

# Installation

    $ go get bitbucket.org/liamstask/goose/lib/goose

# Usage

goose provides several methods to help manage your database schema.

You can access these methods easily using go's built-in flag support:

Example Usage:

```go
var migrate = flag.String("migrate", "", "colon-separated goose db migration command & argument, ie. create:AddTestFieldToTable, down:2, up:-1, down, redo:3")

func main() {
    flag.Parse()

    if *migrate != "" {
        runMigrate(*migrate)
    } else {
        runServerInit()
    }
}

func runMigrate(argList string) {
    args := strings.Split(argList, ":")

    cmd := strings.Title(args[0])
    runArgs := []interface{}{}

    if len(args) > 1 {
        var runArg interface{}
        givenRunCount, err := strconv.ParseInt(args[1], 10, 64)
        if err == nil {
            runArg = interface{}(givenRunCount)
        } else {
            runArg = interface{}(args[1])
        }
        runArgs = append(runArgs, runArg)
    }

    migrationExec := goose.NewMigrationExecutor()

    migrationExec.Do(cmd, runArgs...)
}
```
    $ ./myapp -migrate up
    $ ./myapp -migrate down:3

## Available Methods
    * Do(cmd string, args ...interface{})
    * Create(migrationName string)
    * Up(runCount ...int64)
    * Down(runCount ...int64)
    * Redo(runCount ...int64)
    * Status()

# Configuration

goose expects you to maintain a folder (typically called "db"), which contains the following:

* a dbconf.yml file that describes the database configurations you'd like to use
* a folder called "migrations" which contains .go scripts that implement your migrations

A sample dbconf.yml looks like

    development:
        driver: postgres
        open: user=liam dbname=tester sslmode=disable

Here, `development` specifies the name of the environment, and the `driver` and `open` elements are passed directly to database/sql to access the specified database.
