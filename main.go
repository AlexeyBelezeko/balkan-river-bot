package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// RiverData represents a single river data entry
type RiverData struct {
	River       string
	Station     string
	WaterLevel  string
	WaterChange string
	Discharge   string
	WaterTemp   string
	Tendency    string
}

// DataCache holds the cached water data and its last update time
type DataCache struct {
	data       []RiverData
	lastUpdate time.Time
	mutex      sync.RWMutex
}

// Cache instance
var cache = DataCache{
	data:       nil,
	lastUpdate: time.Time{}, // Zero time
	mutex:      sync.RWMutex{},
}

// GetCachedData retrieves data from cache if fresh, otherwise fetches new data
// Returns the data, last update time, and any error
func GetCachedData() ([]RiverData, time.Time, error) {
	cache.mutex.RLock()
	// Check if we have fresh data (less than 1 hour old)
	if !cache.lastUpdate.IsZero() && time.Since(cache.lastUpdate) < time.Hour {
		log.Printf("Using cached data (last updated: %s)", cache.lastUpdate.Format(time.RFC3339))
		data := cache.data
		lastUpdate := cache.lastUpdate
		cache.mutex.RUnlock()
		return data, lastUpdate, nil
	}
	cache.mutex.RUnlock()

	// Need to update the cache
	log.Printf("Fetching fresh data from website")
	data, err := FetchWaterData()
	if err != nil {
		return nil, time.Time{}, err
	}

	// Update the cache
	cache.mutex.Lock()
	cache.data = data
	cache.lastUpdate = time.Now()
	lastUpdate := cache.lastUpdate
	cache.mutex.Unlock()
	log.Printf("Cache updated with %d entries at %s", len(data), lastUpdate.Format(time.RFC3339))

	return data, lastUpdate, nil
}

// FetchWaterData retrieves water data from the website
func FetchWaterData() ([]RiverData, error) {
	log.Printf("Sending HTTP request to water monitoring website")
	// Send an HTTP GET request to the website
	res, err := http.Get("https://www.hidmet.gov.rs/ciril/osmotreni/stanje_voda.php")
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

	var data []RiverData
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

			data = append(data, RiverData{
				River:       river,
				Station:     station,
				WaterLevel:  waterLevel,
				WaterChange: waterChange,
				Discharge:   discharge,
				WaterTemp:   waterTemp,
				Tendency:    tendencyImg,
			})
		}
	})

	log.Printf("Parsed %d rows, extracted %d valid data entries", rowCount, len(data))
	return data, nil
}

// GetUniqueRivers returns a list of unique river names
func GetUniqueRivers(data []RiverData) []string {
	riverMap := make(map[string]bool)
	for _, entry := range data {
		riverMap[entry.River] = true
	}

	rivers := make([]string, 0, len(riverMap))
	for river := range riverMap {
		rivers = append(rivers, river)
	}

	sort.Strings(rivers)
	log.Printf("Found %d unique rivers", len(rivers))
	return rivers
}

// GetRiverInfo returns information about a specific river
func GetRiverInfo(data []RiverData, riverName string) []RiverData {
	var riverInfo []RiverData
	for _, entry := range data {
		if strings.EqualFold(entry.River, riverName) {
			riverInfo = append(riverInfo, entry)
		}
	}
	log.Printf("Found %d stations for river '%s'", len(riverInfo), riverName)
	return riverInfo
}

// FormatRiverInfo formats river information for display
func FormatRiverInfo(riverData []RiverData, lastUpdate time.Time) string {
	if len(riverData) == 0 {
		return "No information available for this river."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Information for river %s:\n\n", riverData[0].River))

	for _, data := range riverData {
		result.WriteString(fmt.Sprintf("📍 Station: %s\n", data.Station))
		result.WriteString(fmt.Sprintf("💧 Water Level: %s cm\n", data.WaterLevel))
		result.WriteString(fmt.Sprintf("📊 Change: %s cm\n", data.WaterChange))
		result.WriteString(fmt.Sprintf("🌊 Discharge: %s m³/s\n", data.Discharge))
		result.WriteString(fmt.Sprintf("🌡️ Water Temperature: %s °C\n", data.WaterTemp))
		if data.Tendency != "" {
			result.WriteString(fmt.Sprintf("📈 Tendency: %s\n", data.Tendency))
		}
		result.WriteString("\n")
	}

	// Add last update time
	result.WriteString(fmt.Sprintf("🕒 Last update: %s", lastUpdate.Format("2006-01-02 15:04:05")))

	return result.String()
}

func main() {
	// Configure logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Starting Water Bot...")

	// Get the bot token from environment variable
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is not set")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	log.Println("Bot is now listening for messages...")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		// Log incoming messages
		log.Printf("Received message from %s (ID: %d): %s",
			update.Message.From.UserName,
			update.Message.From.ID,
			update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		switch {
		case update.Message.IsCommand():
			switch update.Message.Command() {
			case "start":
				log.Printf("Handling /start command for user %s", update.Message.From.UserName)
				msg.Text = "Welcome to the Water Bot! Use /rivers to see the list of available rivers or /help for more information."
			case "help":
				log.Printf("Handling /help command for user %s", update.Message.From.UserName)
				msg.Text = "Available commands:\n" +
					"/start - Start the bot\n" +
					"/rivers - Show the list of rivers\n" +
					"/river [name] - Show information for a specific river\n" +
					"/help - Show this help message"
			case "rivers":
				log.Printf("Handling /rivers command for user %s", update.Message.From.UserName)
				data, lastUpdate, err := GetCachedData()
				if err != nil {
					msg.Text = "Error fetching water data. Please try again later."
					log.Printf("Error fetching water data: %v", err)
				} else {
					rivers := GetUniqueRivers(data)
					msg.Text = "Available rivers:\n\n"
					for _, river := range rivers {
						msg.Text += "• " + river + "\n"
					}
					msg.Text += "\nUse /river [name] to get detailed information."
					msg.Text += fmt.Sprintf("\n\n🕒 Last update: %s", lastUpdate.Format("2006-01-02 15:04:05"))
				}
			case "river":
				args := update.Message.CommandArguments()
				log.Printf("Handling /river command with args '%s' for user %s", args, update.Message.From.UserName)
				if args == "" {
					msg.Text = "Please specify a river name. Example: /river ДУНАВ"
				} else {
					data, lastUpdate, err := GetCachedData()
					if err != nil {
						msg.Text = "Error fetching water data. Please try again later."
						log.Printf("Error fetching water data: %v", err)
					} else {
						riverInfo := GetRiverInfo(data, args)
						if len(riverInfo) == 0 {
							msg.Text = fmt.Sprintf("No information found for river '%s'. Use /rivers to see the available rivers.", args)
						} else {
							msg.Text = FormatRiverInfo(riverInfo, lastUpdate)
						}
					}
				}
			default:
				log.Printf("Received unknown command /%s from user %s", update.Message.Command(), update.Message.From.UserName)
				msg.Text = "Unknown command. Use /help to see available commands."
			}
		default:
			// Handle non-command messages
			if strings.HasPrefix(update.Message.Text, "/river ") {
				riverName := strings.TrimPrefix(update.Message.Text, "/river ")
				log.Printf("Handling river request for '%s' from user %s", riverName, update.Message.From.UserName)
				data, lastUpdate, err := GetCachedData()
				if err != nil {
					msg.Text = "Error fetching water data. Please try again later."
					log.Printf("Error fetching water data: %v", err)
				} else {
					riverInfo := GetRiverInfo(data, riverName)
					if len(riverInfo) == 0 {
						msg.Text = fmt.Sprintf("No information found for river '%s'. Use /rivers to see the available rivers.", riverName)
					} else {
						msg.Text = FormatRiverInfo(riverInfo, lastUpdate)
					}
				}
			} else {
				log.Printf("Received non-command message from user %s: %s", update.Message.From.UserName, update.Message.Text)
				
				// Fetch data for ГРАДАЦ river as default response
				data, lastUpdate, err := GetCachedData()
				if err != nil {
					// Fallback to default message if error fetching data
					msg.Text = "I don't understand. Use /help to see available commands."
					log.Printf("Error fetching water data: %v", err)
				} else {
					// Get info for ГРАДАЦ river
					riverInfo := GetRiverInfo(data, "ГРАДАЦ")
					
					// Default response with bonus info about ГРАДАЦ
					var response strings.Builder
					response.WriteString("I don't understand. Use /help to see available commands.\n\n")
					response.WriteString("ЈФУИ (Just For Your Information):\n")
					
					if len(riverInfo) > 0 {
						response.WriteString(FormatRiverInfo(riverInfo, lastUpdate))
					} else {
						response.WriteString("No information available for river ГРАДАЦ at the moment.")
					}
					
					msg.Text = response.String()
				}
			}
		}

		log.Printf("Sending response to user %s", update.Message.From.UserName)
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending message: %v", err)
		}
	}
}