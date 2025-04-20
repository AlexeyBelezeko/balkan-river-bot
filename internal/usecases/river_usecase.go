// Package usecases contains the application's business logic
package usecases

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/abelzeko/water-bot/internal/entities"
	"github.com/abelzeko/water-bot/internal/integration"
	"github.com/abelzeko/water-bot/internal/repository"
)

// RiverUseCase handles business logic related to river data
type RiverUseCase struct {
	repo    repository.RiverRepository
	scraper *integration.WaterScraper
}

// NewRiverUseCase creates a new river use case
func NewRiverUseCase(repo repository.RiverRepository, scraper *integration.WaterScraper) *RiverUseCase {
	return &RiverUseCase{
		repo:    repo,
		scraper: scraper,
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

	// Get last update time
	lastUpdate, err := uc.repo.GetLastUpdateTime()
	if err != nil {
		log.Printf("Warning: could not get last update time: %v", err)
	} else {
		log.Printf("Repository updated with %d entries at %s (data timestamp: %s)",
			len(data),
			time.Now().Format(time.RFC3339),
			lastUpdate.Format(time.RFC3339))
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

// GetLastUpdateTime returns when the river data was last updated
func (uc *RiverUseCase) GetLastUpdateTime() (time.Time, error) {
	log.Println("Retrieving last update time")
	return uc.repo.GetLastUpdateTime()
}

// FormatRiverInfo formats river information for display
func (uc *RiverUseCase) FormatRiverInfo(riverData []entities.RiverData, lastUpdate time.Time) string {
	if len(riverData) == 0 {
		return "No information available for this river."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Information for river %s:\n\n", riverData[0].River))

	for _, data := range riverData {
		result.WriteString(fmt.Sprintf("📍 Station: %s\n", data.Station))
		result.WriteString(fmt.Sprintf("💧 Water Level: %s cm\n", data.WaterLevel))

		// Only include fields that have values
		if data.WaterChange != "" {
			result.WriteString(fmt.Sprintf("📊 Change: %s cm\n", data.WaterChange))
		}
		if data.Discharge != "" {
			result.WriteString(fmt.Sprintf("🌊 Discharge: %s m³/s\n", data.Discharge))
		}
		if data.WaterTemp != "" {
			result.WriteString(fmt.Sprintf("🌡️ Water Temperature: %s °C\n", data.WaterTemp))
		}
		if data.Tendency != "" {
			result.WriteString(fmt.Sprintf("📈 Tendency: %s\n", data.Tendency))
		}
		result.WriteString("\n")
	}

	// Add last update time
	result.WriteString(fmt.Sprintf("🕒 Last update: %s", lastUpdate.Format("2006-01-02 15:04:05")))

	return result.String()
}
