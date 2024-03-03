package pkg

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"os"
	"strings"
)

func LoadEnv() (Config, error) {
	result := Config{}

	result.Processor.Key = os.Getenv("NEXTHOS_PROCESSOR_KEY")
	if len(strings.TrimSpace(result.Processor.Key)) == 0 {
		return Config{}, fmt.Errorf("NEXTHOS_PROCESSOR_KEY is required")
	}

	result.Processor.Version = os.Getenv("NEXTHOS_PROCESSOR_VERSION")
	if len(strings.TrimSpace(result.Processor.Version)) == 0 {
		return Config{}, fmt.Errorf("NEXTHOS_PROCESSOR_VERSION is required")
	}

	result.Nats.Url = os.Getenv("NEXTHOS_NATS_URL")
	if len(strings.TrimSpace(result.Nats.Url)) == 0 {
		result.Nats.Url = nats.DefaultURL
	}

	result.Nats.Jwt = os.Getenv("NEXTHOS_NATS_JWT")
	result.Nats.Seed = os.Getenv("NEXTHOS_NATS_SEED")

	return result, nil
}

type Config struct {
	Processor ProcessorConfig
	Nats      NatsConfig
}

type ProcessorConfig struct {
	Key     string
	Version string
}

type NatsConfig struct {
	Url  string
	Jwt  string
	Seed string
}

func (c Config) Connect() (*nats.Conn, error) {
	natsOpts := []nats.Option{
		nats.Name(c.Processor.Key),
	}

	if c.Nats.Jwt != "" {
		natsOpts = append(natsOpts, nats.UserJWTAndSeed(c.Nats.Jwt, c.Nats.Seed))
	}

	return nats.Connect(c.Nats.Url, natsOpts...)
}
