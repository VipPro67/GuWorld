package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/server"
	"github.com/vippro67/GuWorld/backend/internal/worker"
)

func main() {
	// Try loading .env from different possible locations depending on where the server is run from
	var envLoaded bool
	for _, envPath := range []string{".env", "backend/.env", "../.env", "../../.env"} {
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("Loaded environment variables from %s", envPath)
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		log.Println("Warning: No .env file loaded, using default system/environment variables")
	}

	// 1. Connect to Database
	err := db.Connect()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.DB.Close()

	// 2. Initialize Database Schema
	err = db.InitSchema()
	if err != nil {
		log.Fatalf("Database schema initialization failed: %v", err)
	}

	// Initialize Redis Cache
	if err := cache.Connect(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v. Proceeding without caching.", err)
	}

	// 2.2. Load Gu registry cache from database Dictionaries
	err = models.LoadGuRegistry(db.DB)
	if err != nil {
		log.Fatalf("Failed to load Gu registry cache: %v", err)
	}

	// 2.3. Load Item registry cache from database Dictionaries
	err = models.LoadItemRegistry(db.DB)
	if err != nil {
		log.Fatalf("Failed to load Item registry cache: %v", err)
	}

	// 2.5. Start Hunger Decay Background Worker (15-minute intervals for slower, balanced survival gameplay)
	worker.StartHungerWorker(db.DB, 15*time.Minute)

	// 2.6. Start Tribulation background worker
	handlers.StartTribulationWorker()

	// 2.7. Start Mine Monster Worker
	handlers.StartMineMonsterWorker()

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
