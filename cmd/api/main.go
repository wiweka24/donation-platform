package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"

	_ "github.com/jackc/pgx/v5/stdlib"

	"my-platform/internal/handlers"
)

// This struct will hold our loaded configuration
type Config struct {
	DSN        string `mapstructure:"DSN"`
	JWT_SECRET string `mapstructure:"JWT_SECRET"`
}

// This function loads the config.env file from the root folder
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

	// 1. Load Configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	// 2. Connect to the Database
	db, err := sqlx.Connect("pgx", config.DSN)
	if err != nil {
		log.Fatal("cannot connect to database:", err)
	}
	defer db.Close() // Make sure to close the connection when main() exits

	log.Println("Successfully connected to Supabase (PostgreSQL)!")

	// 3. Set up our Gin router
	r := gin.Default()

	// 4. Define a simple test route
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// Create an instance of our AuthHandler, passing it the database connection
	authHandler := handlers.NewAuthHandler(db, config.JWT_SECRET)

	// Group all API routes under /api
	api := r.Group("/api")
	{
		// Group auth routes under /api/auth
		auth := api.Group("/auth")
		{
			// Our new registration endpoint
			auth.POST("/register", authHandler.Register)
			// We will add /login here later
			auth.POST("/login", authHandler.Login)
		}
	}

	// 5. Start the server
	log.Println("Server starting on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("could not start server:", err)
	}
}
