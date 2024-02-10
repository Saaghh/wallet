package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Wallet struct {
	ID           uuid.UUID `json:"id"`
	OwnerID      uuid.UUID `json:"ownerId"`
	Currency     string    `json:"currency"`
	Balance      float64   `json:"balance"`
	CreatedDate  time.Time `json:"createdDate"`
	ModifiedDate time.Time `json:"modifiedDate"`
	Name         string    `json:"name"`
}

type User struct {
	ID      uuid.UUID `json:"id"`
	Email   string    `json:"email"`
	RegDate time.Time `json:"regDate"`
}

type Transaction struct {
	ID             uuid.UUID  `json:"id"`
	CreatedAt      time.Time  `json:"createdAt"`
	AgentWalletID  *uuid.UUID `json:"agentWalletId,omitempty"`
	TargetWalletID *uuid.UUID `json:"targetWalletId,omitempty"`
	Currency       string     `json:"currency"`
	Sum            float64    `json:"sum"`
}

type Transfer struct {
	ID            uuid.UUID
	CreatedAt     time.Time
	AgentWallet   *Wallet
	SumToWithdraw float64
	TargetWallet  *Wallet
	SumToDeposit  float64
}

type UpdateWalletRequest struct {
	Name           *string `json:"name,omitempty"`
	Currency       *string `json:"currency,omitempty"`
	ConversionRate float64 `json:"conversionRate,omitempty"`
}

func (t *Transaction) Validate() error {
	switch {
	case t.Sum == 0:
		return ErrZeroSum
	case t.Sum < 0:
		return ErrNegativeSum
	case t.TargetWalletID == nil:
		return ErrWalletNotFound
	case t.ID == uuid.Nil:
		return ErrNilUUID
	}

	return nil
}

type Claims struct {
	jwt.RegisteredClaims
	UUID uuid.UUID `json:"uuid"`
}
