// Package models defines the data structures for the mock API.
package models

// Building represents a company location/building
type Building struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Address     string `json:"address"`
}

// Room represents a room within a building (internal use, includes sensor_ids)
type Room struct {
	ID         int    `json:"id"`
	BuildingID int    `json:"building_id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Floor      int    `json:"floor"`
	Capacity   int    `json:"capacity"`
	Sensors    []int  `json:"sensor_ids"`
}

// RoomMinimal represents a room without sensor relationships (for chatty API responses)
type RoomMinimal struct {
	ID         int    `json:"id"`
	BuildingID int    `json:"building_id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Floor      int    `json:"floor"`
	Capacity   int    `json:"capacity"`
}

// ToMinimal returns a RoomMinimal without the sensor_ids field
func (r *Room) ToMinimal() *RoomMinimal {
	return &RoomMinimal{
		ID:         r.ID,
		BuildingID: r.BuildingID,
		Name:       r.Name,
		Type:       r.Type,
		Floor:      r.Floor,
		Capacity:   r.Capacity,
	}
}

// ToMinimalSlice converts a slice of Room to a slice of RoomMinimal
func ToMinimalSlice(rooms []Room) []*RoomMinimal {
	result := make([]*RoomMinimal, len(rooms))
	for i := range rooms {
		result[i] = &RoomMinimal{
			ID:         rooms[i].ID,
			BuildingID: rooms[i].BuildingID,
			Name:       rooms[i].Name,
			Type:       rooms[i].Type,
			Floor:      rooms[i].Floor,
			Capacity:   rooms[i].Capacity,
		}
	}
	return result
}

// SensorType defines the type of sensor
type SensorType string

const (
	// Fused sensor types - each sensor returns multiple metrics in one reading
	SensorEnvironment SensorType = "environment" // Returns temperature, humidity, co2
	SensorPower       SensorType = "power"       // Returns voltage, current, power, energy
)

// Sensor represents a sensor in a room
type Sensor struct {
	ID          int        `json:"id"`
	RoomID      int        `json:"room_id"`
	Name        string     `json:"name"`
	Type        SensorType `json:"type"`
	Description string     `json:"description"`
}

// EnvironmentReading represents fused environment sensor readings (temperature, humidity, CO2)
type EnvironmentReading struct {
	SensorID    int     `json:"sensor_id"`
	Temperature float64 `json:"temperature,omitempty"`
	Humidity    float64 `json:"humidity,omitempty"`
	CO2         float64 `json:"co2,omitempty"`
	Timestamp   int64   `json:"timestamp"`
}

// Reading represents a single sensor reading with timestamp
type Reading struct {
	SensorID  int     `json:"sensor_id"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"` // Unix timestamp in seconds
}

// MultiReading represents multiple readings for different metrics (e.g., power: voltage, current, power)
type MultiReading struct {
	SensorID  int     `json:"sensor_id"`
	Voltage   float64 `json:"voltage,omitempty"`
	Current   float64 `json:"current,omitempty"`
	Power     float64 `json:"power,omitempty"`
	Energy    float64 `json:"energy,omitempty"`
	Timestamp int64   `json:"timestamp"` // Unix timestamp in seconds
}

// APIResponse is the standard wrapper for all API responses
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *MetaInfo   `json:"meta,omitempty"`
}

// MetaInfo contains pagination and metadata
type MetaInfo struct {
	Total     int   `json:"total,omitempty"`
	Page      int   `json:"page,omitempty"`
	PerPage   int   `json:"per_page,omitempty"`
	Timestamp int64 `json:"timestamp"` // Unix timestamp in seconds
}

// SensorWithReadings represents a sensor with its readings nested
type SensorWithReadings struct {
	ID          int             `json:"id"`
	RoomID      int             `json:"room_id"`
	Name        string          `json:"name"`
	Type        SensorType      `json:"type"`
	Description string          `json:"description"`
	Readings    []interface{}   `json:"readings"`
}

// RoomWithSensors represents a room with its sensors nested (each sensor has readings)
type RoomWithSensors struct {
	ID         int                `json:"id"`
	BuildingID int                `json:"building_id"`
	Name       string             `json:"name"`
	Type       string             `json:"type"`
	Floor      int                `json:"floor"`
	Capacity   int                `json:"capacity"`
	Sensors    []SensorWithReadings `json:"sensors"`
}

// BuildingWithRoomsNested represents a building with rooms as a nested array
type BuildingWithRoomsNested struct {
	ID          int                  `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Address     string               `json:"address"`
	Rooms       []RoomWithSensors    `json:"rooms"`
}
