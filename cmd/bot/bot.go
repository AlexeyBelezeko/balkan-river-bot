package main

import (
	"log"
	"os"

	"github.com/abelzeko/water-bot/internal/api"
	"github.com/abelzeko/water-bot/internal/integration"
	"github.com/abelzeko/water-bot/internal/integration/openai" // Updated import
	"github.com/abelzeko/water-bot/internal/repository"
	"github.com/abelzeko/water-bot/internal/usecases"
)

func main() {
	// Configure logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Starting Water Bot...")

	// Initialize OpenAI Service
	openAIService, err := openai.NewOpenAIService() // Updated constructor call
	if err != nil {
		log.Fatalf("Failed to initialize OpenAI service: %v", err)
	}

	// Initialize repository
	repo, err := repository.NewSQLiteRiverRepository("")
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	// Initialize scraper
	scraper := integration.NewWaterScraper("")

	// Initialize use case with OpenAI service
	useCase := usecases.NewRiverUseCase(repo, scraper, openAIService)

	// Get the bot token from environment variable
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is not set")
	}

	// Initialize Telegram bot
	telegramBot, err := api.NewTelegramBot(botToken, useCase)
	if err != nil {
		log.Fatalf("Failed to initialize Telegram bot: %v", err)
	}

	// Start the bot
	telegramBot.Start()
}
