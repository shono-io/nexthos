package pkg

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

func NewProcessors(js jetstream.JetStream) (*Processors, error) {
	enc := nats.EncoderForType(nats.JSON_ENCODER)
	kv, err := js.KeyValue(context.Background(), "processors")
	if err != nil {
		return nil, fmt.Errorf("unable to get the processors key value store: %w", err)
	}

	ob, err := js.ObjectStore(context.Background(), "processor_logic")
	if err != nil {
		return nil, fmt.Errorf("unable to get the processor logic object store: %w", err)
	}

	return &Processors{
		enc: enc,
		kv:  kv,
		ob:  ob,
	}, nil
}

type Processor struct {
	Key  string
	Name string
}

type Processors struct {
	enc nats.Encoder
	kv  jetstream.KeyValue
	ob  jetstream.ObjectStore
}

func (p *Processors) Get(ctx context.Context, key string) (*Processor, error) {
	log.Debug().Msgf("getting processor from key %s", key)

	e, err := p.kv.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var proc Processor
	if err := p.enc.Decode("processor", e.Value(), &proc); err != nil {
		return nil, err
	}

	return &proc, nil
}

func (p *Processors) Load(ctx context.Context, key string, version string) (string, error) {
	fqn := fmt.Sprintf("/%s/%s/benthos.yaml", key, version)

	log.Debug().Msgf("loading processor logic from key %s", fqn)
	return p.ob.GetString(ctx, fqn)
}
