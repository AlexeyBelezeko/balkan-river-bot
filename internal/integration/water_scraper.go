// Package integration handles external service interactions
package integration

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/abelzeko/water-bot/internal/entities"
)

// WaterScraper provides functionality to scrape water data from external sources
type WaterScraper struct {
	sourceURL      string
	gradacRiverURL string
}

// NewWaterScraper creates a new water data scraper
func NewWaterScraper(url string) *WaterScraper {
	if url == "" {
		// Default source URL
		url = "https://www.hidmet.gov.rs/ciril/osmotreni/stanje_voda.php"
	}
	return &WaterScraper{
		sourceURL:      url,
		gradacRiverURL: "https://www.hidmet.gov.rs/ciril/osmotreni/nrt_tabela_grafik.php?hm_id=45902&period=7",
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

// FetchGradacRiverData retrieves water data specifically for river ГРАДАЦ
// Only returns valid timestamp-level pairs where level is an integer
func (ws *WaterScraper) FetchGradacRiverData() ([]entities.RiverData, error) {
	log.Printf("Sending HTTP request to fetch river ГРАДАЦ data")
	// Send an HTTP GET request to the special ГРАДАЦ river URL
	res, err := http.Get(ws.gradacRiverURL)
	if err != nil {
		log.Printf("Error fetching ГРАДАЦ river data: %v", err)
		return nil, fmt.Errorf("failed to fetch ГРАДАЦ river data: %v", err)
	}
	defer res.Body.Close()

	// Check for successful response
	if res.StatusCode != 200 {
		log.Printf("Received unexpected status code for ГРАДАЦ river: %d %s", res.StatusCode, res.Status)
		return nil, fmt.Errorf("unexpected status code for ГРАДАЦ river: %d %s", res.StatusCode, res.Status)
	}
	log.Printf("Successfully received HTTP response for ГРАДАЦ river with status: %s", res.Status)

	// Parse the HTML document
	log.Printf("Parsing HTML document for ГРАДАЦ river")
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Printf("Error parsing ГРАДАЦ river HTML: %v", err)
		return nil, fmt.Errorf("failed to parse the ГРАДАЦ river webpage: %v", err)
	}

	var data []entities.RiverData
	processedRows := 0
	validRows := 0
	skippedRows := 0

	// Use UTC for parsing timestamps as the website posts timestamps in UTC
	utc := time.UTC

	// Based on the HTML structure, find all table rows in the document
	// that contain water level data
	doc.Find("table tr").Each(func(index int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() == 2 {
			processedRows++

			// Extract datetime and water level
			dateTimeStr := strings.TrimSpace(cells.Eq(0).Text())
			waterLevelStr := strings.TrimSpace(cells.Eq(1).Text())

			// Skip header rows or rows without proper date format
			if dateTimeStr == "" || dateTimeStr == "Датум и време" ||
				!strings.Contains(dateTimeStr, ".") || !strings.Contains(dateTimeStr, ":") {
				skippedRows++
				return
			}

			// Parse the timestamp in UTC since the website posts timestamps in UTC
			timestamp, parseErr := time.ParseInLocation("02.01.2006 15:04", dateTimeStr, utc)
			if parseErr != nil {
				log.Printf("Warning: Skipping row with invalid timestamp format: %s, %v", dateTimeStr, parseErr)
				skippedRows++
				return
			}

			// Parse water level to verify it's an integer
			waterLevel, parseErr := strconv.Atoi(waterLevelStr)
			if parseErr != nil {
				log.Printf("Warning: Skipping row with non-integer water level: %s", waterLevelStr)
				skippedRows++
				return
			}

			// Only include valid data
			validRows++

			// Create river data entry
			data = append(data, entities.RiverData{
				River:       "ГРАДАЦ",
				Station:     "ДЕГУРИЋ",
				WaterLevel:  fmt.Sprintf("%d", waterLevel), // Ensure it's consistently formatted
				WaterChange: "",                            // Not available in this source
				Discharge:   "",                            // Not available in this source
				WaterTemp:   "",                            // Not available in this source
				Tendency:    "",                            // Not available in this source
				Timestamp:   timestamp,
			})
		}
	})

	log.Printf("ГРАДАЦ river data: processed %d rows, found %d valid entries, skipped %d invalid entries",
		processedRows, validRows, skippedRows)

	// Sorting data by timestamp (oldest first) for consistency
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

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

// FetchRhmzRsData retrieves water data from the novi.rhmzrs.com website
func (ws *WaterScraper) FetchRhmzRsData() ([]entities.RiverData, error) {
	log.Printf("Fetching data from RHMZ RS website")

	// Step 1: Fetch the listing page
	listURL := "https://novi.rhmzrs.com/page/bilten-izvjestaj-o-vodostanju"
	resp, err := http.Get(listURL)
	if err != nil {
		log.Printf("Error fetching RHMZ RS listing page: %v", err)
		return nil, fmt.Errorf("failed to fetch RHMZ RS listing page: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading RHMZ RS listing HTML: %v", err)
		return nil, fmt.Errorf("error reading RHMZ RS listing HTML: %v", err)
	}
	body := string(bodyBytes)

	// Step 2: Extract link to the latest bulletin
	linkRe := regexp.MustCompile(`<a[^>]+href="([^"]+)"[^>]*>Редован\s+хидролошки\s+билтен`)
	match := linkRe.FindStringSubmatch(body)
	if len(match) < 2 {
		log.Printf("Latest RHMZ RS bulletin link not found")
		return nil, fmt.Errorf("latest RHMZ RS bulletin link not found")
	}
	href := match[1]
	if strings.HasPrefix(href, "/") {
		href = "https://novi.rhmzrs.com" + href
	}
	log.Printf("Found bulletin link: %s", href)

	// Step 3: Fetch the bulletin page
	resp2, err := http.Get(href)
	if err != nil {
		log.Printf("Error fetching RHMZ RS bulletin page: %v", err)
		return nil, fmt.Errorf("error fetching RHMZ RS bulletin page: %v", err)
	}
	defer resp2.Body.Close()

	// Step 4: Parse the HTML document using goquery
	doc, err := goquery.NewDocumentFromReader(resp2.Body)
	if err != nil {
		log.Printf("Error parsing RHMZ RS bulletin HTML: %v", err)
		return nil, fmt.Errorf("error parsing RHMZ RS bulletin HTML: %v", err)
	}

	// Step 5: Parse common timestamp
	timestamp := time.Now() // Default timestamp
	doc.Find("table tr").Each(func(i int, tr *goquery.Selection) {
		// Look for the row containing the timestamp text
		if tr.Find("td").Length() > 0 {
			text := strings.TrimSpace(tr.Find("td").First().Text())
			if strings.Contains(text, "НА ДАН") && strings.Contains(text, "ГОДИНЕ") {
				tsRe := regexp.MustCompile(`НА\s+ДАН\s+(\d{2}\.\d{2}\.\d{4})\.\s*ГОДИНЕ,\s*У\s*(\d{1,2}:\d{2})`)
				tsMatch := tsRe.FindStringSubmatch(text)

				if len(tsMatch) == 3 {
					// Parse timestamp from matched date and time
					dateStr := tsMatch[1]
					timeStr := tsMatch[2]
					log.Printf("Extracted RHMZ RS date: '%s', time: '%s'", dateStr, timeStr)

					// Parse timestamp in Serbian/Bosnian time zone
					loc, _ := time.LoadLocation("Europe/Sarajevo")
					t, err := time.ParseInLocation("02.01.2006 15:04", dateStr+" "+timeStr, loc)
					if err == nil {
						timestamp = t
						log.Printf("Successfully parsed RHMZ RS timestamp: %s", timestamp.Format(time.RFC3339))
					} else {
						log.Printf("Error parsing RHMZ RS timestamp: %v", err)
					}
				}
			}
		}
	})

	// Step 6: Extract table data - skip header rows (first few rows with titles)
	var data []entities.RiverData
	var currentRiver string

	// Get table rows (skip header rows)
	var headerPassed bool

	doc.Find("table tr").Each(func(i int, tr *goquery.Selection) {
		cells := tr.Find("td")
		cellCount := cells.Length()

		// Skip rows without enough columns
		if cellCount < 4 {
			return
		}

		// Look for header row that contains column titles
		if !headerPassed {
			headerText := strings.TrimSpace(cells.Eq(0).Text())
			if headerText == "РИЈЕКА" {
				headerPassed = true
				return // Skip this header row
			}
			return // Skip any row before header
		}

		// Check for empty rows or footnote rows
		firstCellText := strings.TrimSpace(cells.Eq(0).Text())
		if firstCellText == "" || strings.Contains(firstCellText, "Напомена") || strings.Contains(firstCellText, "Легенда") {
			return
		}

		// Handle river name - might span multiple rows (use rowspan)
		riverName := strings.TrimSpace(cells.Eq(0).Text())
		if riverName != "" {
			currentRiver = riverName
		} else if currentRiver == "" {
			return // Skip row if no river name is set
		}

		// Extract data from cells
		station := strings.TrimSpace(cells.Eq(1).Text())

		// Skip rows without a station name
		if station == "" {
			return
		}

		// Extract water level (4th column - index 3)
		waterLevelStr := strings.TrimSpace(cells.Eq(3).Text())
		if waterLevelStr == "-" || waterLevelStr == "" {
			waterLevelStr = "0" // Default when no data
		}

		// Extract water change (5th column - index 4)
		waterChange := strings.TrimSpace(cells.Eq(4).Text())

		// Extract water temperature (6th column - index 5)
		waterTemp := strings.TrimSpace(cells.Eq(5).Text())
		if waterTemp == "-" {
			waterTemp = "" // No temperature data
		}

		// Extract discharge (7th column - index 6)
		discharge := strings.TrimSpace(cells.Eq(6).Text())
		if discharge == "-" {
			discharge = "" // No discharge data
		}

		// Extract tendency (8th column - index 7)
		tendency := strings.TrimSpace(cells.Eq(7).Text())
		// Map symbols to our standard format
		switch tendency {
		case "▲":
			tendency = "rising"
		case "▼":
			tendency = "falling"
		case "●":
			tendency = "stable"
		default:
			tendency = ""
		}

		// Create a RiverData entry
		data = append(data, entities.RiverData{
			River:       currentRiver,
			Station:     station,
			WaterLevel:  waterLevelStr,
			WaterChange: waterChange,
			WaterTemp:   waterTemp,
			Discharge:   discharge,
			Tendency:    tendency,
			Timestamp:   timestamp,
		})
	})

	log.Printf("RHMZ RS data: extracted %d river data entries", len(data))
	return data, nil
}
