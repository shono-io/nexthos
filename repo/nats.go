package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
	"github.com/shono-io/nexthos/sdk"
	"strings"
)

func NewNatsRepository(nc *nats.Conn, cfg Config) (Repository, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to jetstream: %w", err)
	}

	ctx := context.Background()
	kv, err := js.KeyValue(ctx, cfg.KeyValueBucket)
	if err != nil {
		return nil, fmt.Errorf("unable to get key value store: %w", err)
	}

	obs, err := js.ObjectStore(ctx, cfg.ObjectStoreBucket)
	if err != nil {
		return nil, fmt.Errorf("unable to get object store: %w", err)
	}

	repo := &natsRepository{
		kv:      kv,
		obs:     obs,
		prefix:  cfg.Prefix,
		updates: make(chan *Change),
		done:    make(chan struct{}),
	}

	return repo, nil
}

type natsRepository struct {
	kv      jetstream.KeyValue
	obs     jetstream.ObjectStore
	prefix  string
	updates chan *Change
	done    chan struct{}
}

func (n *natsRepository) Watch(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	subject := fmt.Sprintf("%s.pipeline.*.version.*", n.prefix)
	log.Info().Msgf("watching for pipeline updates at %q", subject)
	kw, err := n.kv.Watch(ctx, subject)
	if err != nil {
		log.Error().Err(err).Msg("unable to watch for processor updates")
		return
	}

	for {
		select {
		case <-ctx.Done():
			close(n.done)
		case <-n.done:
			if err := kw.Stop(); err != nil {
				log.Warn().Err(err).Msg("unable to stop watching for pipeline updates")
			}
			return
		case msg := <-kw.Updates():
			if msg == nil {
				continue
			}

			var op Operation
			switch msg.Operation() {
			case jetstream.KeyValuePut:
				op = PutOperation
			case jetstream.KeyValuePurge:
				continue
			case jetstream.KeyValueDelete:
				op = DeleteOperation
			}

			var version sdk.PipelineVersion
			if err := json.Unmarshal(msg.Value(), &version); err != nil {
				log.Error().Err(err).Msg("unable to unmarshal stored pipeline version")
				continue
			}

			kp := strings.Split(msg.Key(), ".")
			pipelineId := kp[len(kp)-3]

			n.updates <- &Change{
				Operation:  op,
				PipelineId: pipelineId,
				Version:    version,
				Revision:   msg.Revision(),
			}
		}
	}
}

func (n *natsRepository) Close() error {
	close(n.done)
	return nil
}

func (n *natsRepository) Updates() <-chan *Change {
	return n.updates
}
