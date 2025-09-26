package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize Gin router
	router := gin.Default()

	// Add basic middleware
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Basic health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "randompic-backend",
		})
	})

	// API routes group
	api := router.Group("/api")
	{
		// Placeholder for random images endpoint
		api.GET("/images/random", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Random images endpoint - not implemented yet",
			})
		})
	}

	// Start server on port 8080
	log.Println("Starting server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}