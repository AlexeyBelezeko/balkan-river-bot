// Package entities contains the core domain objects for the water-bot application
package entities

import (
	"time"
)

// RiverData represents a single river data entry in the system
type RiverData struct {
	ID         int64
	River      string    // Name of the river
	Station    string    // Monitoring station name
	WaterLevel string    // Current water level in cm
	WaterTemp  string    // Water temperature in °C
	Timestamp  time.Time // When the data was recorded
}
