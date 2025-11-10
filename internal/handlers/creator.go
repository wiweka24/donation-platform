package handlers

import (
	"log"
	"my-platform/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type CreatorHandler struct {
	DB *sqlx.DB
}

func NewCreatorHandler(db *sqlx.DB) *CreatorHandler {
	return &CreatorHandler{DB: db}
}

func (h *CreatorHandler) GetMyProfile(c *gin.Context) {
	// Get the userID from the context
	userID_any, exists := c.Get("userID")
	if !exists {
		log.Println("UserID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error: UserID not found"})
		return
	}

	userID, ok := userID_any.(int)
	if !ok {
		log.Println("UserID in context is not an int")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error: UserID invalid format"})
		return
	}

	// Fetch the creator profile from the database
	var profile models.Creator
	query := `SELECT id, user_id, username, display_name, widget_secret_token 
            FROM creators WHERE user_id = $1`

	err := h.DB.Get(&profile, query, userID)
	if err != nil {
		log.Println("Failed to get creator profile:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Creator profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *CreatorHandler) GetMyDonations(c *gin.Context) {
	// Get the userID from the context
	userID_any, _ := c.Get("userID")
	userID := userID_any.(int)

	// Fetch the creator's ID from their user_id
	var creator models.Creator
	err := h.DB.Get(&creator, "SELECT id FROM creators WHERE user_id = $1", userID)
	if err != nil {
		log.Println("Failed to find creator for user_id:", userID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Creator profile not found"})
		return
	}

	// Fetch all donations for this creator, newest first
	var donations []models.Donation
	query := `SELECT * FROM donations WHERE creator_id = $1 ORDER BY created_at DESC`
	err = h.DB.Select(&donations, query, creator.ID)
	if err != nil {
		log.Println("Failed to get donations:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch donations"})
		return
	}

	// Return the list
	c.JSON(http.StatusOK, donations)
}
