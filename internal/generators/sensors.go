// Package generators provides mock data generation for the workshop API.
package generators

import (
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/zabbix-workshop/nlld/internal/models"
)

// SensorState tracks the current value and history for each sensor to enable gradual changes
type SensorState struct {
	mu          sync.RWMutex
	current     float64
	lastChange  time.Time
	minValue    float64
	maxValue    float64
	noiseFactor float64 // How much random variation per update
}

// EnvironmentSensorState tracks state for all three environment metrics (temp, humidity, CO2)
type EnvironmentSensorState struct {
	mu           sync.RWMutex
	temp         *SensorState
	humidity     *SensorState
	co2          *SensorState
	lastChange   time.Time
	minTemp      float64
	maxTemp      float64
	minHumidity  float64
	maxHumidity  float64
	minCO2       float64
	maxCO2       float64
}

// Global sensor state management
var (
	sensorStates = make(map[int]*SensorState)           // For power sensors
	envStates    = make(map[int]*EnvironmentSensorState) // For environment sensors
	stateMu      sync.RWMutex
	nowFunc      = time.Now // Overridable for testing
)

func init() {
	// Initialize all sensors with baseline values based on their type and room context
	for i := range sensors {
		sensor := &sensors[i]
		room := GetRoomByID(sensor.RoomID)
		if room == nil {
			continue
		}

		switch sensor.Type {
		case models.SensorEnvironment:
			// Initialize environment sensor state with separate tracking for each metric
			envState := &EnvironmentSensorState{
				minTemp:   getMinTemp(room),
				maxTemp:   getMaxTemp(room),
				minHumidity: getMinHumidity(room),
				maxHumidity: getMaxHumidity(room),
				minCO2:    getMinCO2(room),
				maxCO2:    getMaxCO2(room),
			}

			// Initialize temperature state
			envState.temp = &SensorState{
				current:     calculateInitialTemp(room),
				lastChange:  nowFunc(),
				minValue:    envState.minTemp,
				maxValue:    envState.maxTemp,
				noiseFactor: 0.5, // Temperature changes slowly, ±0.5°C max per reading
			}

			// Initialize humidity state
			envState.humidity = &SensorState{
				current:     calculateInitialHumidity(room),
				lastChange:  nowFunc(),
				minValue:    envState.minHumidity,
				maxValue:    envState.maxHumidity,
				noiseFactor: 3.0, // Humidity can vary more, ±3% max per reading
			}

			// Initialize CO2 state
			envState.co2 = &SensorState{
				current:     calculateInitialCO2(room),
				lastChange:  nowFunc(),
				minValue:    envState.minCO2,
				maxValue:    envState.maxCO2,
				noiseFactor: 80.0, // CO2 varies with occupancy, ±80ppm max per reading
			}

			envState.lastChange = nowFunc()

			stateMu.Lock()
			envStates[sensor.ID] = envState
			stateMu.Unlock()

		case models.SensorPower:
			// Initialize power sensor state (tracks base power value)
			state := &SensorState{
				minValue:    50.0,  // Base power in watts
				maxValue:    200.0, // Max base power
				noiseFactor: 10.0,  // Power variation per update
			}

			// Set initial power value
			state.current = 100.0 + rand.Float64()*50
			state.lastChange = nowFunc()

			stateMu.Lock()
			sensorStates[sensor.ID] = state
			stateMu.Unlock()
		}
	}
}

// Temperature helpers based on room type
func getMinTemp(room *models.Room) float64 {
	switch room.Type {
	case "serverroom":
		return 17.0 // Server rooms are cooler
	case "kitchen":
		return 21.0 // Kitchens warmer
	default:
		return 18.0 // Standard office
	}
}

func getMaxTemp(room *models.Room) float64 {
	switch room.Type {
	case "serverroom":
		return 24.0 // Server rooms kept cool
	case "kitchen":
		return 30.0 // Kitchens can get warm
	default:
		return 26.0 // Standard office max
	}
}

func calculateInitialTemp(room *models.Room) float64 {
	base := 21.0 // Standard office temp
	switch room.Type {
	case "serverroom":
		base = 20.0 // Cooler for servers
	case "kitchen":
		base = 24.0 // Warmer in kitchen
	case "training_room":
		base = 22.0 // Slightly warmer for comfort with people
	}
	return base + rand.Float64()*0.5 - 0.25 // Small random offset
}

// Humidity helpers based on room type
func getMinHumidity(room *models.Room) float64 {
	switch room.Type {
	case "serverroom":
		return 35.0 // Lower humidity for electronics
	default:
		return 30.0
	}
}

func getMaxHumidity(room *models.Room) float64 {
	switch room.Type {
	case "serverroom":
		return 55.0 // Lower humidity for electronics
	default:
		return 75.0
	}
}

func calculateInitialHumidity(room *models.Room) float64 {
	base := 45.0
	switch room.Type {
	case "serverroom":
		base = 45.0 // Controlled for electronics
	case "kitchen":
		base = 60.0 // Higher humidity in kitchen
	}
	return base + rand.Float64()*5 - 2.5
}

// CO2 helpers based on room type
func getMinCO2(room *models.Room) float64 {
	switch room.Type {
	case "kitchen", "training_room":
		return 400.0 // Can start lower in occupied spaces
	default:
		return 450.0
	}
}

func getMaxCO2(room *models.Room) float64 {
	switch room.Type {
	case "kitchen", "training_room":
		return 2500.0 // Higher potential in occupied spaces
	default:
		return 1800.0
	}
}

func calculateInitialCO2(room *models.Room) float64 {
	base := 500.0
	switch room.Type {
	case "training_room", "kitchen":
		base = 700.0 // Higher baseline for occupied spaces
	}
	return base + rand.Float64()*100 - 50
}

// updateValue applies a random walk to the sensor value
func (s *SensorState) update() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only update if at least 5 seconds have passed since last change
	if nowFunc().Sub(s.lastChange) < 5*time.Second {
		return s.current
	}

	// Calculate random delta within noise factor range
	delta := (rand.Float64()*2 - 1) * s.noiseFactor // Random value between -noise and +noise

	// Apply the change
	s.current += delta

	// Clamp to valid range with some headroom
	headroom := s.noiseFactor * 2
	if s.current < s.minValue+headroom {
		s.current = s.minValue + headroom
	}
	if s.current > s.maxValue-headroom {
		s.current = s.maxValue - headroom
	}

	s.lastChange = nowFunc()
	return s.current
}

// GetEnvironmentReading returns fused readings for environment sensors (temp, humidity, CO2)
func GetEnvironmentReading(sensorID int) *models.EnvironmentReading {
	stateMu.RLock()
	envState, exists := envStates[sensorID]
	stateMu.RUnlock()

	if !exists {
		return nil
	}

	envState.mu.Lock()
	defer envState.mu.Unlock()

	// Only update if at least 5 seconds have passed since last change
	if nowFunc().Sub(envState.lastChange) >= 5*time.Second {
		// Update all three metrics
		envState.temp.update()
		envState.humidity.update()
		envState.co2.update()
		envState.lastChange = nowFunc()
	}

	return &models.EnvironmentReading{
		SensorID:    sensorID,
		Temperature: math.Round(envState.temp.current*10) / 10,   // Round to 1 decimal
		Humidity:    math.Round(envState.humidity.current*10) / 10, // Round to 1 decimal
		CO2:         math.Round(envState.co2.current),           // Round to integer
		Timestamp:   nowFunc().Unix(),
	}
}

// GetMultiSensorReading returns readings for power sensors (voltage, current, power, energy)
func GetMultiSensorReading(sensorID int) *models.MultiReading {
	stateMu.RLock()
	state, exists := sensorStates[sensorID]
	stateMu.RUnlock()

	if !exists {
		return nil
	}

	now := nowFunc()
	hour := now.Hour()

	// Power values for smart plugs (simulating typical devices)
	voltage := 230.0 // EU standard voltage

	// Apply time-of-day pattern: higher during business hours (9-18), lower at night
	var loadFactor float64
	switch {
	case hour >= 9 && hour < 18:
		loadFactor = 0.7 + rand.Float64()*0.3 // 70-100% load during work hours
	case hour >= 6 && hour < 9:
		loadFactor = 0.5 + rand.Float64()*0.2 // Morning ramp-up
	case hour >= 18 && hour < 22:
		loadFactor = 0.4 + rand.Float64()*0.2 // Evening wind-down
	default:
		loadFactor = 0.2 + rand.Float64()*0.1 // Night standby mode
	}

	// Apply random walk to power value (gradual changes)
	power := state.update() * loadFactor
	if power < 5 {
		power = 5 // Minimum standby
	}

	current := power / voltage
	energy := float64(now.Unix())/3600 + rand.Float64()*100 // Simulated kWh (hour-based)

	return &models.MultiReading{
		SensorID:  sensorID,
		Voltage:   math.Round(voltage*100) / 100,
		Current:   math.Round(current*1000) / 1000, // mA
		Power:     math.Round(power),
		Energy:    energy,
		Timestamp: now.Unix(),
	}
}

// GetAllSensorReadings returns readings for all sensors in a room
func GetAllSensorReadings(roomID int) []interface{} {
	sensors := GetSensorsByRoom(roomID)
	var readings []interface{}

	for _, sensor := range sensors {
		switch sensor.Type {
		case models.SensorEnvironment:
			if reading := GetEnvironmentReading(sensor.ID); reading != nil {
				readings = append(readings, reading)
			}
		case models.SensorPower:
			if reading := GetMultiSensorReading(sensor.ID); reading != nil {
				readings = append(readings, reading)
			}
		}
	}

	return readings
}
