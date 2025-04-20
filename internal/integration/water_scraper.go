// Package integration handles external service interactions
package integration

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/abelzeko/water-bot/internal/entities"
)

// WaterScraper provides functionality to scrape water data from external sources
type WaterScraper struct {
	sourceURL string
}

// NewWaterScraper creates a new water data scraper
func NewWaterScraper(url string) *WaterScraper {
	if url == "" {
		// Default source URL
		url = "https://www.hidmet.gov.rs/ciril/osmotreni/stanje_voda.php"
	}
	return &WaterScraper{
		sourceURL: url,
	}
}

// FetchWaterData retrieves water data from the website
func (ws *WaterScraper) FetchWaterData() ([]entities.RiverData, error) {
	log.Printf("Sending HTTP request to water monitoring website")
	// Send an HTTP GET request to the website
	res, err := http.Get(ws.sourceURL)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		return nil, fmt.Errorf("failed to fetch the webpage: %v", err)
	}
	defer res.Body.Close()

	// Check for successful response
	if res.StatusCode != 200 {
		log.Printf("Received unexpected status code: %d %s", res.StatusCode, res.Status)
		return nil, fmt.Errorf("unexpected status code: %d %s", res.StatusCode, res.Status)
	}
	log.Printf("Successfully received HTTP response with status: %s", res.Status)

	// Parse the HTML document
	log.Printf("Parsing HTML document")
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		return nil, fmt.Errorf("failed to parse the webpage: %v", err)
	}

	// Extract timestamp from the website
	timestamp := ws.ExtractTimestamp(doc)

	var data []entities.RiverData
	rowCount := 0

	// Iterate over each table row in the document
	doc.Find("table tbody tr").Each(func(index int, row *goquery.Selection) {
		rowCount++
		cells := row.Find("td")
		if cells.Length() >= 10 {
			// Extract river name from the first cell
			river := strings.TrimSpace(cells.Eq(0).Text())

			// Extract station name from the third cell, which contains an <a> tag
			station := strings.TrimSpace(cells.Eq(2).Find("a").Text())

			// Extract water level, water change, discharge, water temperature, and tendency from the respective cells
			waterLevel := strings.TrimSpace(cells.Eq(5).Text())
			waterChange := strings.TrimSpace(cells.Eq(6).Text())
			discharge := strings.TrimSpace(cells.Eq(7).Text())
			waterTemp := strings.TrimSpace(cells.Eq(8).Text())

			// Get tendency image
			tendencyImg := cells.Eq(9).Find("img").AttrOr("alt", "")

			data = append(data, entities.RiverData{
				River:       river,
				Station:     station,
				WaterLevel:  waterLevel,
				WaterChange: waterChange,
				Discharge:   discharge,
				WaterTemp:   waterTemp,
				Tendency:    tendencyImg,
				Timestamp:   timestamp,
			})
		}
	})

	log.Printf("Parsed %d rows, extracted %d valid data entries", rowCount, len(data))
	return data, nil
}

// ExtractTimestamp extracts the timestamp from the HTML document
func (ws *WaterScraper) ExtractTimestamp(doc *goquery.Document) time.Time {
	// Default fallback
	timestamp := time.Now()
	timestampText := ""

	// Look for the timestamp in the page using multiple selectors
	selectors := []string{
		"div.col-md-12",
		"div",
		"h4",
		"div.container",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if strings.Contains(text, "Хидролошки подаци:") {
				log.Printf("Found timestamp text using selector '%s': %s", selector, text)
				timestampText = text
			}
		})
		if timestampText != "" {
			break
		}
	}

	// Parse the timestamp if found
	if timestampText != "" {
		extractedTime := ws.parseTimestampText(timestampText)
		if !extractedTime.IsZero() {
			timestamp = extractedTime
			log.Printf("Successfully extracted timestamp: %s", timestamp.Format(time.RFC3339))
		} else {
			log.Printf("Failed to parse timestamp from: %s", timestampText)
		}
	} else {
		log.Printf("Timestamp text not found, using current time")
	}

	return timestamp
}

// parseTimestampText parses timestamp text from the webpage
func (ws *WaterScraper) parseTimestampText(text string) time.Time {
	// Default fallback
	timestamp := time.Time{}

	// Expected format examples:
	// "Хидролошки подаци: ПЕТАК 18.04.2025. време: 8:00 (06:00 UTC)"
	// "Хидролошки подаци: 18.04.2025. време: 8:00"

	// Try to parse the timestamp
	if strings.Contains(text, "Хидролошки подаци:") && strings.Contains(text, "време:") {
		dateParts := strings.Split(text, "време:")
		if len(dateParts) >= 2 {
			// Extract date part - skip the day name if present
			dateText := strings.TrimSpace(strings.Split(dateParts[0], ":")[1])
			dateFields := strings.Fields(dateText)

			// The date should be in format DD.MM.YYYY.
			// It might be preceded by a day name
			var dateStr string
			for _, field := range dateFields {
				if strings.Contains(field, ".") {
					dateStr = field
					break
				}
			}

			// Extract time part
			timeStr := strings.TrimSpace(strings.Split(dateParts[1], "(")[0])

			log.Printf("Extracted date: '%s', time: '%s'", dateStr, timeStr)

			// Parse date DD.MM.YYYY.
			var day, month, year int
			_, err := fmt.Sscanf(dateStr, "%d.%d.%d.", &day, &month, &year)
			if err != nil {
				log.Printf("Error parsing date from '%s': %v", dateStr, err)
				return timestamp
			}

			// Parse time HH:MM
			var hour, minute int
			_, err = fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
			if err != nil {
				log.Printf("Error parsing time from '%s': %v", timeStr, err)
				return timestamp
			}

			// Create timestamp
			loc, _ := time.LoadLocation("Europe/Belgrade") // Serbian time zone
			timestamp = time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc)
			log.Printf("Successfully parsed timestamp: %s", timestamp.Format(time.RFC3339))
		}
	}

	return timestamp
}
