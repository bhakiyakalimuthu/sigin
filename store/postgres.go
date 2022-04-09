package store

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"time"
)

type Postgres struct {
	db *sqlx.DB
}

func NewPostgres(dbAddress string) *Postgres {
	//source := fmt.Sprintf(
	//	"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
	//	dbAddress, 5432, "test", "postgres", "postgres")
	source := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		dbAddress, 5439, "rpc_endpoint_prod", "endpoint", "e988a3385fF5")
	db := sqlx.MustConnect("postgres", source)
	return &Postgres{
		db: db,
	}
}

func (p *Postgres) Insert(entry []*MethodSignatureEntry) error {
	query := `INSERT INTO rpc_endpoint_eth_method_signatures (id,method_hex_signature,method_text_signature) VALUES (:id,:method_hex_signature,:method_text_signature)`
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	_, err := p.db.NamedExecContext(ctx, query, entry)
	if err != nil {
		return err
	}
	return nil
}
