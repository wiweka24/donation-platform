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

type ProfileResponse struct {
	UserID            int    `db:"user_id" json:"user_id"`
	Username          string `db:"username" json:"username"`
	DisplayName       string `db:"display_name" json:"display_name"`
	WidgetSecretToken string `db:"widget_secret_token" json:"widget_secret_token"`
	Email             string `db:"email" json:"email"`
}

type DonationResponse struct {
	OrderID            string `db:"order_id" json:"order_id"`
	AmountCents        int    `db:"amount_cents" json:"amount_cents"`
	DonorName          string `db:"donor_name" json:"donor_name"`
	DonorMessage       string `db:"donor_message" json:"donor_message"`
	PaymentGatewayTxID string `db:"payment_gateway_tx_id" json:"payment_gateway_tx_id"`
	CreatedAt          string `db:"created_at" json:"created_at"`
	MediaType          string `db:"media_type" json:"media_type"`
	MediaURL           string `db:"media_url" json:"media_url"`
	MediaStartSeconds  int    `db:"media_start_seconds" json:"media_start_seconds"`
	MediaEndSeconds    int    `db:"media_end_seconds" json:"media_end_seconds"`
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
	var profile ProfileResponse
	query := `SELECT 
            c.user_id, c.username, c.display_name, c.widget_secret_token,
            u.email
            FROM creators c
            INNER JOIN users u ON c.user_id = u.id
            WHERE c.user_id = $1`

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
	query_creator := `SELECT id 
                    FROM creators 
                    WHERE user_id = $1`
	err := h.DB.Get(&creator, query_creator, userID)
	if err != nil {
		log.Println("Failed to find creator for user_id:", userID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Creator profile not found"})
		return
	}

	// Fetch all donations for this creator, newest first
	var donations []DonationResponse
	query_donations := `SELECT 
                      order_id, amount_cents, donor_name, donor_message, payment_gateway_tx_id, 
                      created_at, media_type, media_url, media_start_seconds, media_end_seconds
                      FROM donations 
                      WHERE creator_id = $1 AND status = 'settled'
                      ORDER BY created_at DESC`
	err = h.DB.Select(&donations, query_donations, creator.ID)
	if err != nil {
		log.Println("Failed to get donations:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch donations"})
		return
	}

	c.JSON(http.StatusOK, donations)
}
