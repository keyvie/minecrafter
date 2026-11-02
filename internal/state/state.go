package state

import (
	"github.com/uptrace/bun"

	"github.com/keyvie/minecrafter/internal/mojang"
	"github.com/keyvie/minecrafter/internal/rpc"
	"github.com/keyvie/minecrafter/internal/validator"
)

type Public struct {
	Postgres  *bun.DB
	RPC       *rpc.Client
	Mojang    *mojang.Client
	Validator *validator.Validator
}

func New(postgres *bun.DB, rpc *rpc.Client, mojang *mojang.Client, v *validator.Validator) *Public {
	return &Public{
		Postgres:  postgres,
		RPC:       rpc,
		Mojang:    mojang,
		Validator: v,
	}
}
