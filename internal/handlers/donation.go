package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"

	"my-platform/internal/models"
)

type DonationHandler struct {
	DB         *sqlx.DB
	SnapClient snap.Client
	CoreClient coreapi.Client
}

func NewDonationHandler(db *sqlx.DB, serverKey string) *DonationHandler {
	var s snap.Client
	s.New(serverKey, midtrans.Sandbox)

	var c coreapi.Client
	c.New(serverKey, midtrans.Sandbox)

	return &DonationHandler{DB: db, SnapClient: s, CoreClient: c}
}

type CreateDonationRequest struct {
	AmountCents  int    `json:"amount_cents" binding:"required,gt=1000"`
	DonorName    string `json:"donor_name"`
	DonorMessage string `json:"donor_message"`
}

func (h *DonationHandler) CreateDonation(c *gin.Context) {
	username := c.Param("username")

	var req CreateDonationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	var creator models.Creator
	query := `SELECT id FROM creators WHERE username = $1`
	err := h.DB.Get(&creator, query, username)
	if err != nil {
		log.Println("Failed to find creator:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Creator not found"})
		return
	}

	orderID := "DONATION-" + strconv.FormatInt(time.Now().Unix(), 10) + "-C" + strconv.Itoa(creator.ID)

	donorName := req.DonorName
	if donorName == "" {
		donorName = "Anonymous"
	}

	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID,
			GrossAmt: int64(req.AmountCents),
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: donorName,
		},
		Items: &[]midtrans.ItemDetails{
			{
				ID:    "DONATION",
				Price: int64(req.AmountCents),
				Qty:   1,
				Name:  "Donation to " + username,
			},
		},
		CustomField1: strconv.Itoa(creator.ID),
		CustomField2: req.DonorName,
		CustomField3: req.DonorMessage,
	}

	snapResp, err := h.SnapClient.CreateTransaction(snapReq)

	if snapResp == nil {
		log.Println("Failed to create Midtrans transaction (nil response):", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment gateway error."})
		return
	}

	if err != nil {
		log.Printf("Midtrans returned a valid response but also a non-nil error: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Payment link created.",
		"redirect_url": snapResp.RedirectURL,
		"order_id":     orderID,
	})
}

func (h *DonationHandler) HandlePaymentNotification(c *gin.Context) {
	var notification coreapi.TransactionStatusResponse
	if err := c.ShouldBindJSON(&notification); err != nil {
		log.Println("Failed to bind Midtrans notification:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification format"})
		return
	}

	apiResp, err := h.CoreClient.CheckTransaction(notification.OrderID)

	if apiResp == nil {
		log.Println("Failed to verify transaction (nil response) with Midtrans Core API:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found or API error"})
		return
	}
	if err != nil {
		log.Printf("Midtrans Core API returned a valid response but also a non-nil error: %v", err)
	}

	if apiResp.TransactionStatus != "settlement" && apiResp.TransactionStatus != "capture" {
		log.Println("Received non-settled transaction status:", apiResp.TransactionStatus)
		c.JSON(http.StatusOK, gin.H{"status": "ok (not settled)"})
		return
	}

	if apiResp.GrossAmount != notification.GrossAmount {
		log.Println("FRAUD ATTEMPT: GrossAmount mismatch")
		c.JSON(http.StatusForbidden, gin.H{"error": "Amount mismatch"})
		return
	}

	amountFloat, _ := strconv.ParseFloat(apiResp.GrossAmount, 64)
	amountCents := int(amountFloat * 100)

	creatorID, parseErr := strconv.Atoi(notification.CustomField1)
	if parseErr != nil {
		log.Println("Failed to parse creatorID from custom field:", parseErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid custom field data"})
		return
	}

	donorName := notification.CustomField2
	if donorName == "" {
		donorName = "Anonymous"
	}
	donorMessage := notification.CustomField3

	query := `INSERT INTO donations (creator_id, amount_cents, donor_name, donor_message, payment_gateway_tx_id)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (payment_gateway_tx_id) 
            DO NOTHING`

	res, parseErr := h.DB.Exec(query, creatorID, amountCents, donorName, donorMessage, apiResp.TransactionID)
	if parseErr != nil {
		log.Println("Failed to insert donation:", parseErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		log.Println("Duplicate donation notification received, already processed:", apiResp.TransactionID)
		c.JSON(http.StatusOK, gin.H{"status": "ok (duplicate)"})
		return
	}

	log.Printf("SUCCESS: Saved new donation %s for creator %d", apiResp.TransactionID, creatorID)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
