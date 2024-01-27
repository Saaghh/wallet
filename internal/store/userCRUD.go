package store

import (
	"context"
	"fmt"
	"github.com/Saaghh/wallet/internal/model"
)

func (p *Postgres) CreateUser(ctx context.Context, email string) (*model.User, error) {
	user := new(model.User)

	err := p.db.QueryRow(ctx, "INSERT INTO users (email) VALUES ($1) returning id, email, registered_at", email).Scan(&user.ID, &user.Email, &user.RegDate)
	if err != nil {
		return nil, fmt.Errorf("p.db.QueryRow: %w", err)
	}

	return user, nil
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user := new(model.User)

	err := p.db.QueryRow(ctx, "SELECT id, email, registered_at FROM users WHERE email = $1", email).Scan(&user.ID, &user.Email, &user.RegDate)
	if err != nil {
		return nil, fmt.Errorf("p.db.QueryRow: %w", err)
	}

	return user, nil
}
