package main

import (
	"fmt"
	"log"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	if err := db.Connect(); err != nil {
		log.Fatal(err)
	}
	
	globalUUID := "00000000-0000-0000-0000-000000000000"
	channel := "world"
	rows, err := db.DB.Query(`
		SELECT m.id, m.sender_id, p.username, m.content, m.created_at, m.channel
		FROM messages m
		JOIN players p ON m.sender_id = p.id
		WHERE m.receiver_id = $1 AND m.channel = $2
		ORDER BY m.created_at ASC
		LIMIT 50
	`, globalUUID, channel)
	if err != nil {
		log.Fatal("Query error:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, senderID, username, content, channel string
		var createdAt time.Time
		if err := rows.Scan(&id, &senderID, &username, &content, &createdAt, &channel); err != nil {
			log.Println("Scan error:", err)
		} else {
			fmt.Println(id, username, channel)
		}
	}
}
