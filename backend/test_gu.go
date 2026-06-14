package main

import (
	"fmt"
	"log"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func main() {
	err := db.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.DB.Close()

	models.LoadGuDictionary() // Wait, models.LoadGuDictionary is in db, wait.
	
	// We just need to check db.CheckImmortalGuExists
	tx, err := db.DB.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	exists, err := db.CheckImmortalGuExists(tx, "GU_PULLING_MOUNTAIN")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("GU_PULLING_MOUNTAIN exists: %v\n", exists)
}
