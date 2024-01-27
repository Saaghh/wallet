package store

import (
	"context"
	"fmt"
	"github.com/Saaghh/wallet/internal/model"
	"time"
)

func (p *Postgres) CreateUser(ctx context.Context, email string) (int64, error) {
	var userID int64

	err := p.db.QueryRow(ctx, "INSERT INTO users (email, regdate) VALUES ($1, $2) RETURNING id", email, time.Now()).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("p.db.QueryRow: %w", err)
	}

	return userID, nil
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user := new(model.User)

	err := p.db.QueryRow(ctx, "SELECT id, email, regDate FROM users WHERE email = $1", email).Scan(&user.ID, &user.Email, &user.RegDate)
	if err != nil {
		return nil, fmt.Errorf("p.db.QueryRow: %w", err)
	}

	return user, nil
}
