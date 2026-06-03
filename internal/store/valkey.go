package store

import (
	"context"
	"crypto/tls"
	"inst_parser/internal/config"
	"log"

	"github.com/valkey-io/valkey-go"
)

func NewValKeyClient(cfg config.ValKey) valkey.Client {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{cfg.Host},
		Password:    cfg.Password,
		TLSConfig:   &tls.Config{},
		Username:    cfg.Username,
		//DisableCache:      true,
		//ForceSingleClient: true,
		//ConnWriteTimeout:  5 * time.Second,
	})

	if err != nil {
		log.Fatalf("failed to connect to valkey: %v", err)
	}

	ctx := context.Background()

	pingCmd := client.B().Ping().Build()
	_, err = client.Do(ctx, pingCmd).ToString()
	if err != nil {
		log.Fatalf("Ошибка PING: %v", err)
	}

	return client
}
