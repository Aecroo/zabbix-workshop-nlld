// Package main is the entry point for the Zabbix Workshop Mock Data API.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zabbix-workshop/nlld/internal/handlers"
	"github.com/zabbix-workshop/nlld/pkg/config"
)

const serviceFile = `[Unit]
Description=Zabbix Workshop Mock Data API
Documentation=https://github.com/zabbix-workshop/nlld
After=network.target

[Service]
Type=simple
User=%s
WorkingDirectory=%s
ExecStart=%s
Restart=on-failure
RestartSec=5s

# Environment configuration
Environment="API_PORT=8080"
Environment="API_HOST=0.0.0.0"
Environment="DEBUG=false"

# Security hardening
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`

// corsMiddleware adds CORS headers to allow cross-origin requests from Swagger UI
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Handle service management commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--install", "-i":
			installService()
			return
		case "--remove", "-r":
			removeService()
			return
		case "--status", "-s":
			checkServiceStatus()
			return
		case "--help", "-h":
			printHelp()
			return
		}
	}

	cfg := config.LoadFromEnv()

	handler := handlers.NewAPIHandler()
	swaggerHandler := handlers.NewSwaggerHandler()

	// Create a custom mux to wrap all routes with CORS
	mux := http.NewServeMux()

	// Swagger UI endpoint - /swagger/
	mux.Handle("/swagger/", swaggerHandler)

	// Root endpoint - API documentation
	mux.HandleFunc("/api", handler.RootHandler)

	// All data endpoint - /api/all
	mux.HandleFunc("/api/all", handler.AllHandler)

	// Buildings endpoints - handle /api/buildings and /api/buildings/{id} and /api/buildings/{id}/rooms
	mux.HandleFunc("/api/buildings/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/buildings/")

		if path == "" || path == "/" {
			handler.BuildingsHandler(w, r)
		} else {
			// /api/buildings/{id} or /api/buildings/{id}/rooms
			parts := strings.Split(path, "/")

			if len(parts) == 0 || parts[0] == "" {
				http.NotFound(w, r)
				return
			}

			idStr := parts[0]
			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.NotFound(w, r)
				return
			}

			if len(parts) >= 2 && parts[1] == "rooms" {
				handler.RoomsByBuildingHandler(w, r, id)
			} else {
				handler.BuildingByIDHandler(w, r, id)
			}
		}
	})

	// Rooms endpoints - handle /api/rooms/{id} and /api/rooms/{id}/sensors
	mux.HandleFunc("/api/rooms/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/rooms/")

		if path == "" || path == "/" {
			http.NotFound(w, r)
			return
		}

		parts := strings.Split(path, "/")

		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}

		idStr := parts[0]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if len(parts) >= 2 && parts[1] == "sensors" {
			// Only allow /api/rooms/{id}/sensors - reject any deeper paths like /readings
			if len(parts) > 2 {
				http.NotFound(w, r)
				return
			}
			handler.SensorsByRoomHandler(w, r, id)
			return
		}

		handler.RoomByIDHandler(w, r, id)
	})

	// Sensors readings endpoint - /api/sensors/{id}/readings
	mux.HandleFunc("/api/sensors/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/sensors/")

		if path == "" || path == "/" {
			http.NotFound(w, r)
			return
		}

		parts := strings.Split(path, "/")

		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}

		idStr := parts[0]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		handler.SensorReadingsHandler(w, r, id)
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("Starting Zabbix Workshop Mock Data API on %s", addr)
	log.Printf("API Endpoints:")
	log.Printf("  GET /api              - List available endpoints")
	log.Printf("  GET /api/all          - Get all buildings, rooms, sensors, and readings (easy mode)")
	log.Printf("  GET /api/buildings    - List all buildings")
	log.Printf("  GET /api/buildings/1  - Get building by ID")
	log.Printf("  GET /api/buildings/1/rooms - List rooms in building")
	log.Printf("  GET /api/rooms/1      - Get room details")
	log.Printf("  GET /api/rooms/1/sensors - List sensors in room")
	log.Printf("  GET /api/sensors/1/readings - Get sensor readings (individual)")
	log.Printf("  GET /swagger/         - OpenAPI Swagger UI documentation")

	// Wrap the mux with CORS middleware and serve
	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Zabbix Workshop Mock Data API

Usage:
  api [command]

Commands:
  --install, -i    Install systemd service and start the API
  --remove, -r     Remove systemd service and stop the API
  --status, -s     Check service status
  --help, -h       Show this help message

Without a command, starts the API server directly.

Examples:
  api                    # Start the API server
  api --install          # Install as systemd service
  sudo api --remove      # Remove systemd service (requires root)
  api --status           # Check if service is running`)
}

func installService() {
	// Get current user
	username := os.Getenv("USER")
	if username == "" {
		username = "root"
	}

	// Get binary path
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}

	// Get working directory (parent of bin/)
	workDir := filepath.Dir(filepath.Dir(binaryPath))

	// Generate service file content
	serviceContent := fmt.Sprintf(serviceFile, username, workDir, binaryPath)

	// Write service file
	servicePath := "/etc/systemd/system/zabbix-workshop-api.service"
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing service file: %v\n", err)
		fmt.Fprintln(os.Stderr, "Please run with sudo to install the service.")
		os.Exit(1)
	}

	fmt.Println("Service file installed successfully.")

	// Reload systemd and start service
	cmd := exec.Command("systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reloading systemd: %v\n%s\n", err, output)
		os.Exit(1)
	}

	cmd = exec.Command("systemctl", "enable", "zabbix-workshop-api.service")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling service: %v\n%s\n", err, output)
		os.Exit(1)
	}

	cmd = exec.Command("systemctl", "start", "zabbix-workshop-api.service")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting service: %v\n%s\n", err, output)
		os.Exit(1)
	}

	fmt.Println("Service enabled and started successfully.")
	fmt.Println("\nCheck status with:")
	fmt.Println("  systemctl status zabbix-workshop-api.service")
	fmt.Println("\nView logs with:")
	fmt.Println("  journalctl -u zabbix-workshop-api.service -f")
}

func removeService() {
	// Stop the service
	cmd := exec.Command("systemctl", "stop", "zabbix-workshop-api.service")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error stopping service: %v\n%s\n", err, output)
	}

	// Disable the service
	cmd = exec.Command("systemctl", "disable", "zabbix-workshop-api.service")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error disabling service: %v\n%s\n", err, output)
	}

	// Remove service file
	servicePath := "/etc/systemd/system/zabbix-workshop-api.service"
	if err := os.Remove(servicePath); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing service file: %v\n", err)
		os.Exit(1)
	}

	// Reload systemd
	cmd = exec.Command("systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reloading systemd: %v\n%s\n", err, output)
		os.Exit(1)
	}

	fmt.Println("Service removed successfully.")
}

func checkServiceStatus() {
	cmd := exec.Command("systemctl", "is-active", "zabbix-workshop-api.service")
	output, err := cmd.CombinedOutput()
	status := strings.TrimSpace(string(output))

	if err != nil {
		fmt.Printf("Service is not running: %s\n", status)
		return
	}

	fmt.Printf("Service status: %s\n", status)

	// Show additional info
	cmd = exec.Command("systemctl", "status", "zabbix-workshop-api.service")
	if output, err := cmd.CombinedOutput(); err == nil {
		fmt.Println("\nDetailed status:")
		fmt.Print(string(output))
	}
}
