// Package usecases contains the application's business logic
package usecases

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/abelzeko/water-bot/internal/entities"
	"github.com/abelzeko/water-bot/internal/integration"
	"github.com/abelzeko/water-bot/internal/integration/openai"
	"github.com/abelzeko/water-bot/internal/repository"
)

// RiverUseCase handles business logic related to river data
type RiverUseCase struct {
	repo          repository.RiverRepository
	scraper       *integration.WaterScraper
	openAIService openai.OpenAIService
}

// NewRiverUseCase creates a new river use case
func NewRiverUseCase(repo repository.RiverRepository, scraper *integration.WaterScraper, openAIService openai.OpenAIService) *RiverUseCase {
	return &RiverUseCase{
		repo:          repo,
		scraper:       scraper,
		openAIService: openAIService,
	}
}

// RefreshRiverData fetches fresh data and updates the repository
func (uc *RiverUseCase) RefreshRiverData() error {
	log.Println("Starting river data refresh process...")

	// Fetch main water data from external source
	data, err := uc.scraper.FetchWaterData()
	if err != nil {
		return fmt.Errorf("failed to fetch general water data: %v", err)
	}
	log.Printf("Successfully fetched %d river data entries", len(data))

	// Fetch ГРАДАЦ river data
	gradacData, err := uc.scraper.FetchGradacRiverData()
	if err != nil {
		log.Printf("Warning: failed to fetch ГРАДАЦ river data: %v", err)
		// Continue with the main data if ГРАДАЦ fetch fails
	} else {
		log.Printf("Successfully fetched %d ГРАДАЦ river data entries", len(gradacData))
		// Append ГРАДАЦ data to the main data set
		data = append(data, gradacData...)
	}

	// Fetch RHMZ RS data
	rhmzRsData, err := uc.scraper.FetchRhmzRsData()
	if err != nil {
		log.Printf("Warning: failed to fetch RHMZ RS data: %v", err)
		// Continue with the main data if RHMZ RS fetch fails
	} else {
		log.Printf("Successfully fetched %d RHMZ RS data entries", len(rhmzRsData))
		// Append RHMZ RS data to the main data set
		data = append(data, rhmzRsData...)
	}

	// Save all data to repository
	if err := uc.repo.SaveRiverData(data); err != nil {
		return fmt.Errorf("failed to save data to repository: %v", err)
	}

	return nil
}

// GetRiverDataByName retrieves data for a specific river
func (uc *RiverUseCase) GetRiverDataByName(riverName string) ([]entities.RiverData, error) {
	log.Printf("Retrieving data for river: %s", riverName)
	return uc.repo.GetRiverDataByName(riverName)
}

// GetAvailableRivers returns a list of all river names
func (uc *RiverUseCase) GetAvailableRivers() ([]string, error) {
	log.Println("Retrieving list of available rivers")
	return uc.repo.GetUniqueRivers()
}

// HandleNaturalLanguageQuery interprets a user's free-text query using the AI service
// and returns an appropriate response string.
func (uc *RiverUseCase) HandleNaturalLanguageQuery(ctx context.Context, query string) (string, error) {
	log.Printf("Interpreting natural language query: %s", query)

	rivers, err := uc.GetAvailableRivers()
	if err != nil {
		log.Printf("Error fetching available rivers: %v", err)
		return "Sorry, I couldn't fetch the list of rivers right now.", nil
	}

	// Call the OpenAI service to interpret the query
	agentResp, err := uc.openAIService.InterpretUserQuery(ctx, query, rivers)
	if err != nil {
		log.Printf("Error interpreting user query via OpenAI: %v", err)
		// Return a generic error message for the user
		return "Sorry, I'm having trouble understanding right now. Please try again later or use /help.", nil
	}

	log.Printf("Agent response: Command='%s', River='%s', Message='%s'",
		agentResp.CommandName, agentResp.SerbianRiverName, agentResp.UserMessage)

	// Process the agent's response
	switch agentResp.CommandName {
	case "GetRiverDataByName":
		if agentResp.SerbianRiverName != "" {
			// Agent identified intent and river name, fetch and format data
			log.Printf("Agent identified river: %s. Fetching data...", agentResp.SerbianRiverName)
			riverData, err := uc.GetRiverDataByName(agentResp.SerbianRiverName)
			if err != nil {
				log.Printf("Error fetching river data after agent interpretation: %v", err)
				return "Sorry, I couldn't fetch the data for that river right now.", nil
			}
			if len(riverData) == 0 {
				// Combine agent's confirmation (if any) with 'not found' message
				msg := agentResp.UserMessage
				if msg != "" {
					msg += "\n\n"
				}
				msg += fmt.Sprintf("However, I couldn't find any information for river '%s'. Use /rivers to see available ones.", agentResp.SerbianRiverName)
				return msg, nil
			}
			// Combine agent's confirmation (if any) with the formatted data
			msg := agentResp.UserMessage
			if msg != "" {
				msg += "\n\n"
			}
			msg += uc.FormatRiverInfo(riverData)
			return msg, nil
		} else {
			// Agent identified intent but not a specific river, use the agent's message
			log.Printf("Agent identified intent GetRiverDataByName but no specific river found.")
			// Return the agent's message (e.g., "Which river?")
			return agentResp.UserMessage, nil
		}
	case "GeneralQuery":
		// Agent determined it's a general query, just return the generated message
		log.Printf("Agent identified general query.")
		return agentResp.UserMessage, nil
	default:
		// Fallback if agent returns an unexpected command or empty response
		log.Printf("Agent returned unexpected command: %s", agentResp.CommandName)
		return "I'm not sure how to respond to that. You can use /help for commands.", nil
	}
}

// FormatRiverInfo formats river information for display
func (uc *RiverUseCase) FormatRiverInfo(riverData []entities.RiverData) string {
	if len(riverData) == 0 {
		return "No information available for this river."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Information for river %s:\n\n", riverData[0].River))

	for _, data := range riverData {
		result.WriteString(fmt.Sprintf("📍 Station: %s\n", data.Station))
		result.WriteString(fmt.Sprintf("💧 Water Level: %s cm\n", data.WaterLevel))

		// Only include fields that have values
		if data.WaterTemp != "" {
			result.WriteString(fmt.Sprintf("🌡️ Water Temperature: %s °C\n", data.WaterTemp))
		}

		result.WriteString(fmt.Sprintf("🕒 Last update: %s", data.Timestamp.Format("2006-01-02 15:04:05 MST")))

		result.WriteString("\n\n")
	}

	// Add last update time with timezone

	return result.String()
}
