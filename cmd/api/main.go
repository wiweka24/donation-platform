package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"

	_ "github.com/jackc/pgx/v5/stdlib"

	"my-platform/internal/handlers"
	"my-platform/internal/middleware"
)

// This struct will hold our loaded configuration
type Config struct {
	DSN                 string `mapstructure:"DSN"`
	JWT_SECRET          string `mapstructure:"JWT_SECRET"`
	MIDTRANS_SERVER_KEY string `mapstructure:"MIDTRANS_SERVER_KEY"`
}

// Function loads the config.env file from the root folder
func loadConfig() (config Config, err error) {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}

func main() {
	log.Println("Starting donation platform server...")

	// Load Configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	// Connect to the Database
	db, err := sqlx.Connect("pgx", config.DSN)
	if err != nil {
		log.Fatal("cannot connect to database:", err)
	}
	defer db.Close()
	log.Println("Successfully connected to Supabase (PostgreSQL)!")

	// Set up our Gin router
	r := gin.Default()

	// Simple test route
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// Create an instance o the handler
	authHandler := handlers.NewAuthHandler(db, config.JWT_SECRET)
	creatorHandler := handlers.NewCreatorHandler(db)
	donationHandler := handlers.NewDonationHandler(db, config.MIDTRANS_SERVER_KEY)

	// All API routes under /api
	api := r.Group("/api")
	{
		// Auth Endpoint
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Protected Endpoint
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(config.JWT_SECRET))
		{
			protected.GET("/me", creatorHandler.GetMyProfile)
		}

		api.POST("/webhook/payment", donationHandler.HandlePaymentNotification)
		api.POST("/donate/:username", donationHandler.CreateDonation)
	}

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("could not start server:", err)
	}
}
