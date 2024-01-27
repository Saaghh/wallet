package model

import "time"

type Wallet struct {
	ID           int64
	OwnerID      int64
	Currency     string
	Balance      float64
	CreatedDate  time.Time
	ModifiedDate time.Time
}

type User struct {
	ID      int64
	Email   string
	RegDate time.Time
}
