// Package entities contains the core domain objects for the water-bot application
package entities

import (
	"time"
)

// RiverData represents a single river data entry in the system
type RiverData struct {
	ID          int64
	River       string    // Name of the river
	Station     string    // Monitoring station name
	WaterLevel  string    // Current water level in cm
	WaterChange string    // Change in water level in cm
	Discharge   string    // Water discharge in m³/s
	WaterTemp   string    // Water temperature in °C
	Tendency    string    // Tendency indicator (rising, falling, stable)
	Timestamp   time.Time // When the data was recorded
}
