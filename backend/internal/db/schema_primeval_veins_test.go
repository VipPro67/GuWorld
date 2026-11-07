package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"
)

// Use an isolated PostgreSQL test database; each case rolls back its own schema.
func TestPrimevalVeinsLegacyMigration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for PostgreSQL migration regression")
	}
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	for _, scenario := range []string{"fresh", "legacy", "archive_failure"} {
		t.Run(scenario, func(t *testing.T) {
			tx, err := database.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			exec := func(query string) {
				t.Helper()
				if _, err := tx.Exec(query); err != nil {
					t.Fatal(err)
				}
			}
			exec(`CREATE SCHEMA vein_regression; SET LOCAL search_path TO vein_regression;
   CREATE TABLE regions (id VARCHAR(50) PRIMARY KEY); INSERT INTO regions VALUES ('r');`)
			exec(primevalVeinsSchema)
			if scenario == "fresh" {
				exec(primevalVeinsSchema)
				return
			}
			exec(`ALTER TABLE primeval_veins DROP CONSTRAINT primeval_veins_region_id_x_y_key;
   INSERT INTO primeval_veins (id,region_id,x,y,rarity,created_at,expires_at,produced_minutes) VALUES
   ('00000000-0000-0000-0000-000000000001','r',1,1,'Hiếm',NOW()-INTERVAL '3 days',NOW()-INTERVAL '1 day',13),
   ('00000000-0000-0000-0000-000000000002','r',1,1,'Thường',NOW()-INTERVAL '2 days',NOW()+INTERVAL '1 day',7),
   ('00000000-0000-0000-0000-000000000003','r',1,1,'Sử Thi',NOW()-INTERVAL '2 days',NOW()+INTERVAL '2 days',19),
   ('00000000-0000-0000-0000-000000000004','r',2,2,'Thường',NOW(),NOW()+INTERVAL '1 day',3);
   CREATE TEMP TABLE original_veins AS SELECT id,to_jsonb(v) AS row_data FROM primeval_veins v;
   SAVEPOINT before_repair;`)
			if scenario == "archive_failure" {
				// An archive collision must fail closed and restore every source row.
				exec(`INSERT INTO primeval_vein_duplicate_archive (id,row_data)
    SELECT id,row_data FROM original_veins WHERE id='00000000-0000-0000-0000-000000000001';`)
				if _, err := tx.Exec(primevalVeinsSchema); err == nil {
					t.Fatal("expected archive collision")
				}
				exec(`ROLLBACK TO SAVEPOINT before_repair;`)
				var count int
				if err := tx.QueryRow(`SELECT COUNT(*) FROM primeval_veins`).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != 4 {
					t.Fatalf("rollback retained %d rows, want 4", count)
				}
				return
			}
			exec(primevalVeinsSchema)
			exec(primevalVeinsSchema)
			var live, archived, unchanged int
			if err := tx.QueryRow(`SELECT COUNT(*) FROM primeval_veins`).Scan(&live); err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRow(`SELECT COUNT(*) FROM primeval_vein_duplicate_archive`).Scan(&archived); err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRow(`SELECT COUNT(*) FROM original_veins o JOIN (
   SELECT id,to_jsonb(v) AS row_data FROM primeval_veins v
   UNION ALL SELECT id,row_data FROM primeval_vein_duplicate_archive
  ) preserved USING(id,row_data)`).Scan(&unchanged); err != nil {
				t.Fatal(err)
			}
			if live != 2 || archived != 2 || unchanged != 4 {
				t.Fatalf("live=%d archive=%d unchanged=%d", live, archived, unchanged)
			}
			var keeper string
			if err := tx.QueryRow(`SELECT id::text FROM primeval_veins WHERE x=1 AND y=1`).Scan(&keeper); err != nil {
				t.Fatal(err)
			}
			if keeper != "00000000-0000-0000-0000-000000000002" {
				t.Fatalf("unexpected keeper %s", keeper)
			}
			result, err := tx.Exec(`INSERT INTO primeval_veins(region_id,x,y,rarity,expires_at)
   VALUES('r',1,1,'Thường',NOW()+INTERVAL '1 day') ON CONFLICT(region_id,x,y) DO NOTHING`)
			if err != nil {
				t.Fatal(err)
			}
			if count, err := result.RowsAffected(); err != nil || count != 0 {
				t.Fatalf("occupied tile result=%d err=%v", count, err)
			}
		})
	}
}

// A harvest must wait at the table lock before it can take any vein row lock.
func TestPrimevalVeinsMigrationBlocksHarvestLocks(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for PostgreSQL migration regression")
	}
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	schema := fmt.Sprintf("vein_lock_%d", time.Now().UnixNano())
	if _, err := database.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	defer database.Exec("DROP SCHEMA " + schema + " CASCADE")
	setup, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer setup.Rollback()
	if _, err := setup.Exec("SET LOCAL search_path TO " + schema + "; CREATE TABLE regions (id VARCHAR(50) PRIMARY KEY);" + primevalVeinsSchema); err != nil {
		t.Fatal(err)
	}
	if err := setup.Commit(); err != nil {
		t.Fatal(err)
	}
	migration, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer migration.Rollback()
	if _, err := migration.Exec("SET LOCAL search_path TO " + schema + ";" + primevalVeinsSchema); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	rows, err := database.QueryContext(ctx, "SELECT id FROM "+schema+".primeval_veins FOR UPDATE NOWAIT")
	if rows != nil {
		rows.Close()
	}
	if err == nil || ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("harvest bypassed migration table lock: query error=%v context=%v", err, ctx.Err())
	}
}
