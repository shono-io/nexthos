package pkg

import (
	"context"
	"encoding/json"
	"github.com/shono-io/mini"
	"github.com/shono-io/nexthos/exec"
	"github.com/shono-io/nexthos/repo"
)

type Config struct {
	Repository repo.Config
	Executor   exec.Config
}

func NewWorker() *Worker {
	return &Worker{}
}

type Worker struct {
	service    *mini.Service
	configChan <-chan []byte
}

func (w *Worker) Init(service *mini.Service, configChan <-chan []byte) error {
	w.service = service
	w.configChan = configChan
	return nil
}

func (w *Worker) Run(ctx context.Context) error {
	var workerCancel context.CancelFunc
	var workerContext context.Context
	var err error

	for {
		select {
		case <-ctx.Done():
			if workerCancel != nil {
				workerCancel()
			}

			return nil

		case b := <-w.configChan:
			if b == nil {
				if workerCancel != nil {
					workerCancel()
				}

				return nil
			}

			var cfg Config
			err = json.Unmarshal(b, &cfg)
			if err != nil {
				if workerCancel != nil {
					workerCancel()
				}

				return err
			}

			// -- close the worker execution
			if workerCancel != nil {
				workerCancel()
			}

			// -- create a new worker execution
			workerContext, workerCancel = context.WithCancel(ctx)
			go w.perform(workerContext, cfg)
		}
	}
}

func (w *Worker) Close() {
}

func (w *Worker) perform(ctx context.Context, cfg Config) {
	w.service.Log.Debug().Msg("initializing repository")
	r, err := repo.NewNatsRepository(w.service.Nats(), cfg.Repository)
	if err != nil {
		w.service.Log.Err(err).Msg("unable to create repository")
		return
	}

	w.service.Log.Debug().Msg("initializing executor")
	e, err := exec.NewDockerExecutor(cfg.Executor)
	if err != nil {
		w.service.Log.Err(err).Msg("unable to create worker execution")
		return
	}

	go r.Watch(ctx)

	w.service.Log.Info().Msg("ready to receive pipeline updates")
	for {
		select {
		case <-ctx.Done():
			w.service.Log.Info().Msg("worker execution stopped")

			if err := r.Close(); err != nil {
				w.service.Log.Warn().Err(err).Msg("unable to close repository")
			}

			e.Close()
			return

		case ru := <-r.Updates():
			if ru == nil {
				continue
			}

			w.service.Log.Debug().Str("p_id", ru.PipelineId).Str("p_version", ru.Version.Version).Msg("pipeline update received")

			w.handleUpdate(context.Background(), ru)

		case fb := <-e.Feedback():
			if fb == nil {
				continue
			}

			w.service.Log.Debug().Str("p_id", fb.PipelineId).Str("p_version", fb.PipelineVersion).Msg("execution update received")

			w.handleFeedback(context.Background(), fb)
		}
	}
}

func (w *Worker) handleUpdate(ctx context.Context, ru *repo.Change) {
	switch ru.Operation {
	case repo.PutOperation:
		w.service.Log.Info().Str("p_id", ru.PipelineId).Str("p_version", ru.Version.Version).Msg("pipeline updated")
		//if err := a.executor.Ensure(ctx, ru.Data); err != nil {
		//    log.Error().Err(err).Msg("unable to ensure pipeline")
		//    return
		//}

	case repo.DeleteOperation:
		w.service.Log.Info().Str("p_id", ru.PipelineId).Str("p_version", ru.Version.Version).Msg("pipeline removed")

		//if err := a.executor.Ensure(ctx, ru.Data); err != nil {
		//    log.Error().Err(err).Msg("unable to ensure pipeline deleted")
		//    return
		//}
	}
}

func (w *Worker) handleFeedback(ctx context.Context, fb *exec.Feedback) {
	w.service.Log.Info().
		Str("p_id", fb.PipelineId).
		Str("p_version", fb.PipelineVersion).
		Str("exec_id", fb.ExecutionId).
		Str("action", fb.Action).
		Msg("execution updated")
}
