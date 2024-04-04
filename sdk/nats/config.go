package nats

import (
	"github.com/nats-io/nats.go"
	"os"
	"strings"
)

const (
	UrlEnvVar  = "NATS_URL"
	JwtEnvVar  = "NATS_JWT"
	SeedEnvVar = "NATS_SEED"
)

func FromEnv(name string, prefix string) nats.Option {
	if !strings.HasSuffix(prefix, "_") {
		prefix = prefix + "_"
	}

	return func(options *nats.Options) error {
		jwt := os.Getenv(prefix + JwtEnvVar)
		seed := os.Getenv(prefix + SeedEnvVar)
		if jwt != "" && seed != "" {
			if err := nats.UserJWTAndSeed(jwt, seed)(options); err != nil {
				return err
			}
		}

		options.Name = name

		return nil
	}
}

func GetUrl(prefix string) string {
	return os.Getenv(prefix + UrlEnvVar)
}
