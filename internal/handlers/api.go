// Package handlers provides HTTP request handling for the mock API.
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/zabbix-workshop/nlld/internal/generators"
	"github.com/zabbix-workshop/nlld/internal/models"
)

// APIHandler handles all API requests
type APIHandler struct{}

// NewAPIHandler creates a new API handler instance
func NewAPIHandler() *APIHandler {
	return &APIHandler{}
}

// writeJSONResponse writes a JSON response with the given status code
func (h *APIHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeErrorResponse writes an error response
func (h *APIHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := models.APIResponse{
		Success: false,
		Error:   message,
		Meta:    &models.MetaInfo{Timestamp: time.Now().Unix()},
	}
	h.writeJSONResponse(w, statusCode, response)
}

// RootHandler handles GET /api - returns available endpoints
func (h *APIHandler) RootHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	endpoints := map[string]string{
		"GET /api/all":              "Get all buildings, rooms, sensors, and readings (easy mode)",
		"GET /api/buildings":        "List all buildings/locations",
		"GET /api/buildings/{id}":   "Get building details by ID",
		"GET /api/buildings/{id}/rooms":  "List rooms in a building",
		"GET /api/rooms/{id}":       "Get room details by ID",
		"GET /api/rooms/{id}/sensors":   "List sensors in a room",
		"GET /api/sensors/{id}/readings": "Get latest sensor readings (individual sensor only)",
	}

	response := models.APIResponse{
		Success: true,
		Data:    endpoints,
		Meta: &models.MetaInfo{
			Timestamp: time.Now().Unix(),
		},
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

// BuildingsHandler handles GET /api/buildings - list all buildings
func (h *APIHandler) BuildingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	buildings := generators.GetBuildings()
	response := models.APIResponse{
		Success: true,
		Data:    buildings,
		Meta: &models.MetaInfo{
			Total:     len(buildings),
			Timestamp: time.Now().Unix(),
		},
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

// BuildingByIDHandler handles GET /api/buildings/{id} - get building by ID
func (h *APIHandler) BuildingByIDHandler(w http.ResponseWriter, r *http.Request, id int) {
	building := generators.GetBuildingByID(id)
	if building == nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Building not found")
		return
	}

	response := models.APIResponse{
		Success: true,
		Data:    building,
		Meta: &models.MetaInfo{
			Timestamp: time.Now().Unix(),
		},
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

// RoomsByBuildingHandler handles GET /api/buildings/{id}/rooms - list rooms in a building
func (h *APIHandler) RoomsByBuildingHandler(w http.ResponseWriter, r *http.Request, buildingID int) {
	rooms := generators.GetRoomsByBuilding(buildingID)
	// Return minimal room data without sensor_ids for chatty API pattern
	minimalRooms := models.ToMinimalSlice(rooms)
	response := models.APIResponse{
		Success: true,
		Data:    minimalRooms,
		Meta: &models.MetaInfo{
			Total:     len(minimalRooms),
			Page:      1,
			PerPage:   len(minimalRooms),
			Timestamp: time.Now().Unix(),
		},
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

// RoomByIDHandler handles GET /api/rooms/{id} - get room details by ID
func (h *APIHandler) RoomByIDHandler(w http.ResponseWriter, r *http.Request, id int) {
	room := generators.GetRoomByID(id)
	if room == nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Room not found")
		return
	}

	// Return minimal room data without sensor_ids for chatty API pattern
	response := models.APIResponse{
		Success: true,
		Data:    room.ToMinimal(),
		Meta: &models.MetaInfo{
			Timestamp: time.Now().Unix(),
		},
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

// SensorsByRoomHandler handles GET /api/rooms/{id}/sensors - list sensors in a room
func (h *APIHandler) SensorsByRoomHandler(w http.ResponseWriter, r *http.Request, roomID int) {
	sensors := generators.GetSensorsByRoom(roomID)
	response := models.APIResponse{
		Success: true,
		Data:    sensors,
		Meta: &models.MetaInfo{
			Total:     len(sensors),
			Timestamp: time.Now().Unix(),
		},
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

// SensorReadingsHandler handles GET /api/sensors/{id}/readings - get latest sensor readings
func (h *APIHandler) SensorReadingsHandler(w http.ResponseWriter, r *http.Request, sensorID int) {
	sensor := generators.GetSensorByID(sensorID)
	if sensor == nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Sensor not found")
		return
	}

	var reading interface{}
	switch sensor.Type {
	case models.SensorEnvironment:
		reading = generators.GetEnvironmentReading(sensorID)
	case models.SensorPower:
		reading = generators.GetMultiSensorReading(sensorID)
	default:
		h.writeErrorResponse(w, http.StatusBadRequest, "Unknown sensor type")
		return
	}

	if reading == nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate sensor reading")
		return
	}

	response := models.APIResponse{
		Success: true,
		Data:    reading,
		Meta: &models.MetaInfo{
			Timestamp: time.Now().Unix(),
		},
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

// AllHandler handles GET /api/all - returns all buildings, rooms, sensors, and readings
func (h *APIHandler) AllHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	buildings := generators.GetBuildings()

	// Build nested structure: data.[building].rooms.[room].sensors.[sensor].readings
	result := make([]models.BuildingWithRoomsNested, 0, len(buildings))
	totalSensors := 0

	for _, building := range buildings {
		rooms := generators.GetRoomsByBuilding(building.ID)

		// Build rooms as array with nested sensors (each sensor has its readings)
		roomsWithData := make([]models.RoomWithSensors, 0, len(rooms))

		for _, room := range rooms {
			sensors := generators.GetSensorsByRoom(room.ID)
			allReadings := generators.GetAllSensorReadings(room.ID)

			totalSensors += len(sensors)

			// Build map of sensor readings for quick lookup
			readingsMap := make(map[int]interface{})
			for _, reading := range allReadings {
				switch v := reading.(type) {
				case *models.EnvironmentReading:
					readingsMap[v.SensorID] = v
				case *models.MultiReading:
					readingsMap[v.SensorID] = v
				}
			}

			// Build sensors with their readings nested inside each sensor
			sensorsWithReadings := make([]models.SensorWithReadings, 0, len(sensors))
			for _, sensor := range sensors {
				readings := []interface{}{readingsMap[sensor.ID]}

				sensorsWithReadings = append(sensorsWithReadings, models.SensorWithReadings{
					ID:          sensor.ID,
					RoomID:      sensor.RoomID,
					Name:        sensor.Name,
					Type:        sensor.Type,
					Description: sensor.Description,
					Readings:    readings,
				})
			}

			roomsWithData = append(roomsWithData, models.RoomWithSensors{
				ID:         room.ID,
				BuildingID: room.BuildingID,
				Name:       room.Name,
				Type:       room.Type,
				Floor:      room.Floor,
				Capacity:   room.Capacity,
				Sensors:    sensorsWithReadings,
			})
		}

		result = append(result, models.BuildingWithRoomsNested{
			ID:          building.ID,
			Name:        building.Name,
			Description: building.Description,
			Address:     building.Address,
			Rooms:       roomsWithData,
		})
	}

	response := models.APIResponse{
		Success: true,
		Data:    result,
		Meta: &models.MetaInfo{
			Total:     totalSensors,
			Timestamp: time.Now().Unix(),
		},
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}
