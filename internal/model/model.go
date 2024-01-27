package model

import "time"

type Wallet struct {
	id           int64
	ownerID      int64
	currency     string
	balance      float64
	createdDate  time.Time
	modifiedDate time.Time
}

type User struct {
	ID      int64
	Email   string
	RegDate time.Time
}
