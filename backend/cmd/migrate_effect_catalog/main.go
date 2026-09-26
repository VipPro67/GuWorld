// Command migrate_effect_catalog performs the guarded database migration after
// InitSchema has installed the staging columns and snapshot tables.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/vippro67/GuWorld/backend/internal/db"
)

func main() {
	rollback := flag.Int64("rollback", 0, "restore the exact snapshot for this revision")
	constraints := flag.Bool("install-constraints", false, "install and validate block-only checks after a successful backfill")
	flag.Parse()
	if err := db.Connect(); err != nil {
		fatal(err)
	}
	defer db.DB.Close()
	if err := db.InitSchema(); err != nil {
		fatal(err)
	}
	if *constraints {
		if err := db.InstallEffectBlockV2Constraints(context.Background()); err != nil {
			fatal(err)
		}
		fmt.Println("installed and validated Block v2 constraints")
		return
	}
	if *rollback > 0 {
		if err := db.RollbackEffectMigration(context.Background(), *rollback); err != nil {
			fatal(err)
		}
		fmt.Printf("rolled back effect catalog revision %d; restart the backend before serving traffic\n", *rollback)
		return
	}
	result, err := db.BackfillEffectsBlockV2(context.Background())
	if err != nil {
		for _, conflict := range result.Conflicts {
			fmt.Fprintf(os.Stderr, "%s %s: %s\n", conflict.Table, conflict.Key, conflict.Reason)
		}
		fatal(err)
	}
	fmt.Printf("validated revision %d: updated=%d unchanged=%d; restart the backend before serving traffic\n", result.Revision, result.Updated, result.Skipped)
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
