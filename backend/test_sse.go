package main

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	db.Connect()
	defer db.DB.Close()
	
	var playerID uuid.UUID
	var username string
	err := db.DB.QueryRow("SELECT id, username FROM players LIMIT 1").Scan(&playerID, &username)
	if err != nil { log.Fatal(err) }
	
	token, _ := auth.GenerateToken(playerID, username)
	fmt.Println(token)
}
