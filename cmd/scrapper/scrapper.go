// filepath: /Users/abelezeko/Projects/water-bot/cmd/scrapper/scrapper.go
package main

import (
	"log"
	"os"

	"github.com/abelzeko/water-bot/internal/integration"
	"github.com/abelzeko/water-bot/internal/repository"
	"github.com/abelzeko/water-bot/internal/usecases"
	"github.com/robfig/cron/v3"
)

func main() {
	// Configure logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Starting Water Bot Scraper...")

	// Initialize repository
	repo, err := repository.NewSQLiteRiverRepository("")
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	// Initialize scraper
	scraper := integration.NewWaterScraper("")

	// Initialize use case
	useCase := usecases.NewRiverUseCase(repo, scraper)

	// Run use case immediately on startup
	if err := useCase.RefreshRiverData(); err != nil {
		log.Printf("Initial data refresh failed: %v", err)
	}

	// Set up cron scheduler to run every hour
	c := cron.New()
	_, err = c.AddFunc("0 * * * *", func() {
		if err := useCase.RefreshRiverData(); err != nil {
			log.Printf("Scheduled data refresh failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to set up cron job: %v", err)
	}

	log.Println("Scraper has been scheduled to run hourly")
	c.Start()

	// Keep the program running
	select {}
}
