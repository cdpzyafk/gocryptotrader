package tests

import (
	"fmt"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	mg "github.com/thrasher-corp/gocryptotrader/database/migration"
	"github.com/thrasher-corp/gocryptotrader/database/repository/audit"
	auditPSQL "github.com/thrasher-corp/gocryptotrader/database/repository/audit/postgres"
	auditSQlite "github.com/thrasher-corp/gocryptotrader/database/repository/audit/sqlite"
)

func TestAudit(t *testing.T) {
	testCases := []struct {
		name   string
		config database.Config
		audit  audit.Repository
		runner func(t *testing.T)
		closer func(t *testing.T, dbConn *database.Database) error
		output interface{}
	}{
		{
			"SQLite",
			database.Config{
				Driver:            "sqlite",
				ConnectionDetails: drivers.ConnectionDetails{Database: path.Join(tempDir, "./testdb.db")},
			},
			auditSQlite.Audit(),
			writeAudit,
			closeDatabase,
			nil,
		},
		{
			"Postgres",
			postgresTestDatabase,
			auditPSQL.Audit(),
			writeAudit,
			nil,
			nil,
		},
	}

	for _, tests := range testCases {
		test := tests

		t.Run(test.name, func(t *testing.T) {

			mg.MigrationDir = filepath.Join("../migration", "migrations")

			if !checkValidConfig(t, &test.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := connectToDatabase(t, &test.config)

			if err != nil {
				t.Fatal(err)
			}

			mLogger := mg.MLogger{}
			migrations := mg.Migrator{
				Log: mLogger,
			}

			migrations.Conn = dbConn

			err = migrations.LoadMigrations()
			if err != nil {
				t.Fatal(err)
			}

			err = migrations.RunMigration()
			if err != nil {
				t.Fatal(err)
			}

			if test.audit != nil {
				audit.Audit = test.audit
			}

			if test.runner != nil {
				test.runner(t)
			}

			switch v := test.output.(type) {

			case error:
				if v.Error() != test.output.(error).Error() {
					t.Fatal(err)
				}
				return
			default:
				break
			}

			if test.closer != nil {
				err = test.closer(t, dbConn)
				if err != nil {
					t.Log(err)
				}
			}
		})
	}
}

func writeAudit(t *testing.T) {
	t.Helper()
	var wg sync.WaitGroup

	for x := 0; x < 200; x++ {
		wg.Add(1)

		go func(x int) {
			defer wg.Done()
			test := fmt.Sprintf("test-%v", x)
			audit.Event(test, test, test)
		}(x)
	}

	wg.Wait()
}