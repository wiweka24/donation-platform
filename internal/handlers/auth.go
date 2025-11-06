package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	"my-platform/internal/models" // Import our models package
)

// AuthHandler will hold the database connection
type AuthHandler struct {
	DB        *sqlx.DB
	JwtSecret string
}

// NewAuthHandler creates a new handler with the DB connection
func NewAuthHandler(db *sqlx.DB, jwtSecret string) *AuthHandler {
	return &AuthHandler{DB: db, JwtSecret: jwtSecret}
}

// RegisterRequest defines the JSON struct we expect from the client
type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	Username    string `json:"username" binding:"required,min=3"`
	DisplayName string `json:"display_name" binding:"required"`
}

// Register is the handler function for creator registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest

	// 1. Validate the incoming JSON
	// Gin's 'ShouldBindJSON' uses the 'binding' tags in our struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 2. Hash the password
	// We MUST NOT store the plain-text password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Password hashing error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error, please try again."})
		return
	}

	// 3. TODO: Generate a unique widget_secret_token
	// For now, we'll use a simple placeholder.
	// Later, we'll use a secure random string (e.g., from a UUID library)
	widgetToken := "temp_token_" + req.Username

	// 4. Create the user and creator profile in a database transaction
	// A transaction ensures that *both* tables are updated, or neither are.
	tx, err := h.DB.Beginx()
	if err != nil {
		log.Println("Failed to begin transaction:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error."})
		return
	}

	// Ensure the transaction is rolled back on error
	defer tx.Rollback()

	// Step 4a: Insert into 'users' table
	var newUser models.User
	userQuery := `INSERT INTO users (email, password_hash) 
	              VALUES ($1, $2) 
				  			RETURNING id, email, created_at, updated_at`

	err = tx.Get(&newUser, userQuery, req.Email, string(passwordHash))
	if err != nil {
		log.Println("Failed to insert new user:", err)
		// This will fail if the email is already taken
		c.JSON(http.StatusConflict, gin.H{"error": "Email or username may already be in use."})
		return
	}

	// Step 4b: Insert into 'creators' table
	creatorQuery := `INSERT INTO creators (user_id, username, display_name, widget_secret_token)
	                 VALUES ($1, $2, $3, $4)`

	_, err = tx.Exec(creatorQuery, newUser.ID, req.Username, req.DisplayName, widgetToken)
	if err != nil {
		log.Println("Failed to insert new creator profile:", err)
		// This will fail if the username is already taken
		c.JSON(http.StatusConflict, gin.H{"error": "Email or username may already be in use."})
		return
	}

	// 5. Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Println("Failed to commit transaction:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error."})
		return
	}

	// 6. Send a successful response
	// We don't send the password hash back, just a success message.
	c.JSON(http.StatusCreated, gin.H{
		"message":  "User created successfully.",
		"user_id":  newUser.ID,
		"email":    newUser.Email,
		"username": req.Username,
	})
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) createJWT(user models.User) (string, error) {
	// Create the claims
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour * 24 * 7).Unix(),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with jwt secret
	tokenString, err := token.SignedString([]byte(h.JwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	var user models.User
	query := `SELECT id, email, password_hash FROM users WHERE email = $1`
	err := h.DB.Get(&user, query, req.Email)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password."})
			return
		}

		log.Println("Database error on login:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error."})
		return
	}

  // Compare stored passwordHash with the user entered password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password."})
		return
	}

	tokenString, err := h.createJWT(user)
	if err != nil {
		log.Println("Failed to create JWT:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error."})
		return
	}

  // Response
	c.JSON(http.StatusOK, gin.H{"message": "Login successful.", "token": tokenString})
}
