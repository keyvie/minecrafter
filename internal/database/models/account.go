package models

import (
	"time"

	"github.com/uptrace/bun"
)

const AccountTable = "accounts"

type Account struct {
	bun.BaseModel `bun:"table:accounts"`

	UUID      string    `bun:"uuid,pk"`
	UserID    string    `bun:"user_id"`
	CreatedAt time.Time `bun:"created_at"`
}
