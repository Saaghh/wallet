package model

import (
	"time"
)

type Wallet struct {
	ID           int64     `json:"id"`
	OwnerID      int64     `json:"ownerId"`
	Currency     string    `json:"currency"`
	Balance      float64   `json:"balance"`
	CreatedDate  time.Time `json:"createdDate"`
	ModifiedDate time.Time `json:"modifiedDate"`
	Name         string    `json:"name"`
}

type User struct {
	ID      int64     `json:"id"`
	Email   string    `json:"email"`
	RegDate time.Time `json:"regDate"`
}

type Transaction struct {
	ID             int64     `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	AgentWalletID  *int64    `json:"agentWalletId,omitempty"`
	TargetWalletID *int64    `json:"targetWalletId,omitempty"`
	Currency       string    `json:"currency"`
	Sum            float64   `json:"sum"`
}

type UpdateWalletRequest struct {
	Name     *string `json:"name,omitempty"`
	Currency *string `json:"currency,omitempty"`
}
