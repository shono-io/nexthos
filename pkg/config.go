package pkg

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"os"
	"strings"
)

func LoadEnv() (Config, error) {
	result := Config{}

	result.Context = os.Getenv("SHONO_CONTEXT")
	if len(strings.TrimSpace(result.Context)) == 0 {
		return Config{}, fmt.Errorf("SHONO_CONTEXT is required")
	}

	result.Processor.Key = os.Getenv("SHONO_PROCESSOR_KEY")
	if len(strings.TrimSpace(result.Processor.Key)) == 0 {
		return Config{}, fmt.Errorf("SHONO_PROCESSOR_KEY is required")
	}

	result.Processor.Version = os.Getenv("SHONO_PROCESSOR_VERSION")
	if len(strings.TrimSpace(result.Processor.Version)) == 0 {
		return Config{}, fmt.Errorf("SHONO_PROCESSOR_VERSION is required")
	}

	result.Shono.Url = os.Getenv("SHONO_URL")
	if len(strings.TrimSpace(result.Shono.Url)) == 0 {
		result.Shono.Url = nats.DefaultURL
	}

	result.Shono.Jwt = os.Getenv("SHONO_JWT")
	result.Shono.Seed = os.Getenv("SHONO_SEED")

	return result, nil
}

type Config struct {
	Context string

	Processor ProcessorConfig
	Shono     ShonoConfig
}

type ProcessorConfig struct {
	Key     string
	Version string
}

type ShonoConfig struct {
	Url  string
	Jwt  string
	Seed string
}

func (c Config) Connect() (*nats.Conn, error) {
	natsOpts := []nats.Option{
		nats.Name(c.Processor.Key),
		nats.UserJWTAndSeed(c.Shono.Jwt, c.Shono.Seed),
	}

	return nats.Connect(c.Shono.Url, natsOpts...)
}
