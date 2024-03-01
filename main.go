package main

import (
	"context"
	"fmt"
	benthos "github.com/benthosdev/benthos/v4/public/service"
	"github.com/nats-io/nats.go/jetstream"
	services "github.com/nats-io/nats.go/micro"
	"github.com/shono-io/nexthos/pkg"

	_ "github.com/benthosdev/benthos/v4/public/components/io"
	_ "github.com/benthosdev/benthos/v4/public/components/pure"
	_ "github.com/benthosdev/benthos/v4/public/components/pure/extended"
)

func main() {
	ctx := context.Background()

	cfg, err := pkg.LoadEnv()
	if err != nil {
		panic(err)
	}

	nc, err := cfg.Connect()
	if err != nil {
		panic(fmt.Errorf("unable to connect to data cluster: %w", err))
	}

	js, err := jetstream.New(nc)
	if err != nil {
		panic(fmt.Errorf("unable to connect to data jetstream: %w", err))
	}

	procs, err := pkg.NewProcessors(js)
	if err != nil {
		panic(err)
	}

	proc, err := procs.Get(ctx, cfg.Processor.Key)
	if err != nil {
		panic(err)
	}

	procContent, err := procs.Load(ctx, proc.Key, cfg.Processor.Version)
	if err != nil {
		panic(err)
	}

	fmt.Println(procContent)
	//log.Debug().Msgf("loaded processor logic: %s", procContent)

	sb := benthos.NewStreamBuilder()
	if err := sb.SetYAML(procContent); err != nil {
		panic(fmt.Errorf("unable to load yaml content: %w", err))
	}

	// request handler
	echoHandler := func(req services.Request) {
		req.Respond(req.Data())
	}

	fmt.Println("Starting echo service")

	_, err = services.AddService(nc, services.Config{
		Name:    proc.Key,
		Version: cfg.Processor.Version,
		// base handler
		Endpoint: &services.EndpointConfig{
			Subject: fmt.Sprintf("svc.%s.%s", cfg.Context, proc.Key),
			Handler: services.HandlerFunc(echoHandler),
		},
	})

	if err != nil {
		panic(err)
	}

	stream, err := sb.Build()
	if err != nil {
		panic(err)
	}

	if err := stream.Run(ctx); err != nil {
		panic(err)
	}

	<-ctx.Done()
}
