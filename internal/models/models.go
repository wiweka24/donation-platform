package models

import "time"

// We use 'db' tags for sqlx to automatically map
// the database column names (snake_case) to our Go fields (CamelCase).

// User represents a user's authentication details.
type User struct {
	ID           int       `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// Creator represents a creator's public profile and settings.
type Creator struct {
	ID                int       `db:"id"`
	UserID            int       `db:"user_id"`
	Username          string    `db:"username"`
	DisplayName       string    `db:"display_name"`
	WidgetSecretToken string    `db:"widget_secret_token"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// Donation represents a single completed donation.
type Donation struct {
	ID                  int       `db:"id"`
	CreatorID           int       `db:"creator_id"`
	AmountCents         int       `db:"amount_cents"`
	DonorName           string    `db:"donor_name"`
	DonorMessage        string    `db:"donor_message"`
	PaymentGatewayTxID  string    `db:"payment_gateway_tx_id"`
	CreatedAt           time.Time `db:"created_at"`
}