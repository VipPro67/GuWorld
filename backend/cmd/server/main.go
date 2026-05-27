package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/server"
	"github.com/vippro67/GuWorld/backend/internal/worker"
)

func main() {
	// Load environment variables (from backend/.env)
	err := godotenv.Load(".env")
	if err != nil {
		// Try loading from parent or relative path if running from different directory
		_ = godotenv.Load()
	}

	// 1. Connect to Database
	err = db.Connect()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.DB.Close()

	// 2. Initialize Database Schema
	err = db.InitSchema()
	if err != nil {
		log.Fatalf("Database schema initialization failed: %v", err)
	}

	// 2.2. Load Gu registry cache from database Dictionaries
	err = models.LoadGuRegistry(db.DB)
	if err != nil {
		log.Fatalf("Failed to load Gu registry cache: %v", err)
	}

	// 2.5. Start Hunger Decay Background Worker (15-minute intervals for slower, balanced survival gameplay)
	worker.StartHungerWorker(db.DB, 15*time.Minute)

	// 3. Setup router & run
	r := server.SetupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Cultivation Backend listening on port %s", port)
	err = r.Run(":" + port)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
