package goose

import (
	"testing"
)

func TestBasics(t *testing.T) {

	dbconf, err := NewDBConf("../../db-sample", "test")
	if err != nil {
		t.Fatal(err)
	}

	got := []string{dbconf.MigrationsDir, dbconf.Env, dbconf.Driver.Name, dbconf.Driver.OpenStr}
	want := []string{"../../db-sample/migrations", "test", "postgres", "user=adamssarah dbname=tester sslmode=disable"}

	for i, s := range got {
		if s != want[i] {
			t.Errorf("Unexpected DBConf value. got %v, want %v", s, want[i])
		}
	}
}
