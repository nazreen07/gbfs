package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Struct to represent the feed URLs from the GBFS response
type GBFSFeed struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Struct for the main GBFS response
type GBFSMainResponse struct {
	Data struct {
		EN struct {
			Feeds []GBFSFeed `json:"feeds"`
		} `json:"en"`
	} `json:"data"`
}

// Struct for the free bike status response
type FreeBikeStatus struct {
	Data struct {
		Bikes []struct {
			BikeID string `json:"bike_id"`
		} `json:"bikes"`
	} `json:"data"`
}

// Struct for provider information, including only Location and URL
type Provider struct {
	Location string
	URL      string
}

// Create Prometheus gauges for each provider's bike availability
var providerBikes = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "available_bikes",
		Help: "Number of bikes available from providers",
	},
	[]string{"location", "url"},
)

// Create a Prometheus gauge for total available bikes across all providers
var totalBikesGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "total_available_bikes",
		Help: "Total number of bikes available across all providers",
	},
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(providerBikes)
	prometheus.MustRegister(totalBikesGauge)
}

// Function to retrieve provider details from environment variables
func getProvidersFromEnv() ([]Provider, error) {
	var providers []Provider

	for i := 1; ; i++ {
		locationKey := "provider" + strconv.Itoa(i) + "_region"
		urlKey := "provider" + strconv.Itoa(i) + "_url"

		location := os.Getenv(locationKey)
		url := os.Getenv(urlKey)

		// Break loop if no more provider entries
		if location == "" && url == "" {
			break
		}

		// Only add provider if both fields are present
		if location != "" && url != "" {
			providers = append(providers, Provider{
				Location: location,
				URL:      url,
			})
		}
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers found in environment variables")
	}
	return providers, nil
}

// Function to fetch the free bike status URL from the main GBFS feed
func fetchFreeBikeStatusURL(gbfsMainURL string) (string, error) {
	resp, err := http.Get(gbfsMainURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var gbfsMain GBFSMainResponse
	if err := json.NewDecoder(resp.Body).Decode(&gbfsMain); err != nil {
		return "", err
	}

	// Loop through the feeds to find the "free_bike_status" URL
	for _, feed := range gbfsMain.Data.EN.Feeds {
		if feed.Name == "free_bike_status" {
			return feed.URL, nil
		}
	}

	return "", fmt.Errorf("free_bike_status not found in %s", gbfsMainURL)
}

// Function to fetch and parse the free bike status data
func fetchFreeBikeStatusData(freeBikeStatusURL string) (int, error) {
	resp, err := http.Get(freeBikeStatusURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Parse the response into the FreeBikeStatus struct
	var freeBikeStatus FreeBikeStatus
	if err := json.Unmarshal(body, &freeBikeStatus); err != nil {
		return 0, err
	}

	// Return the number of bikes
	return len(freeBikeStatus.Data.Bikes), nil
}

// Function to fetch data and update Prometheus metrics
func ingestGBFSData() {
	providers, err := getProvidersFromEnv()
	if err != nil {
		log.Printf("Error retrieving providers from environment: %v", err)
		return
	}

	totalBikes := 0

	// Fetch and update Prometheus metrics for each provider
	for _, provider := range providers {
		// Step 1: Fetch the free_bike_status URL from the provider
		freeBikeStatusURL, err := fetchFreeBikeStatusURL(provider.URL)
		if err != nil {
			log.Printf("Error fetching free bike status URL from %s: %v", provider.URL, err)
			continue
		}

		// Step 2: Fetch the number of available bikes
		numBikes, err := fetchFreeBikeStatusData(freeBikeStatusURL)
		if err != nil {
			log.Printf("Error fetching free bike status data from %s: %v", freeBikeStatusURL, err)
			continue
		}

		// Log the bike availability for each provider
		fmt.Printf("Provider Location: %s, Available Bikes: %d\n", provider.Location, numBikes)

		// Update the Prometheus gauge for this provider
		providerBikes.With(prometheus.Labels{
			"location": provider.Location,
			"url":      provider.URL,
		}).Set(float64(numBikes))

		totalBikes += numBikes
	}

	// Update the total available bikes gauge
	totalBikesGauge.Set(float64(totalBikes))

	// Log the total number of bikes available
	fmt.Printf("Total Available Bikes: %d\n", totalBikes)

	log.Printf("Ingested data for %d providers. Total bikes available: %d", len(providers), totalBikes)
}

// Background Goroutine to automate ingestion every 5 minutes
func startAutomatedIngestion() {
	go func() {
		for {
			// Run the ingestion process
			ingestGBFSData()
			// Wait for 5 minutes before the next ingestion
			time.Sleep(5 * time.Minute)
		}
	}()
}

func main() {
	// Start automated ingestion in the background
	startAutomatedIngestion()

	// Create a new Gin router
	router := gin.Default()

	// Define the API route for manual ingestion (optional)
	router.POST("/ingest", func(c *gin.Context) {
		ingestGBFSData()
		c.String(http.StatusOK, "Manual ingestion complete")
	})

	// Expose Prometheus metrics on /metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Run the server on port 8080
	router.Run(":8080")
}
