package exec

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"github.com/shono-io/nexthos/sdk"
)

const (
	DefaultImage = "jeffail/benthos:latest"
)

func NewDockerExecutor(cfg Config) (Executor, error) {
	d := &docker{
		feedback: make(chan *Feedback),
		Image:    DefaultImage,
		clientOpts: []client.Opt{
			client.WithAPIVersionNegotiation(),
		},
	}

	if cfg.FromEnv {
		d.clientOpts = append(d.clientOpts, client.FromEnv)
	} else {
		if cfg.Url != "" {
			d.clientOpts = append(d.clientOpts, client.WithHost(cfg.Url))
		}
	}

	dc, err := client.NewClientWithOpts(d.clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create docker client: %w", err)
	}
	d.dc = dc

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		f := filters.NewArgs()
		f.Add("type", "container")
		options := types.EventsOptions{
			Filters: f,
		}

		evts, errs := dc.Events(ctx, options)

		// -- out first task after starting to listen, we will need to retrieve the current state of the
		// -- docker environment. This is because we will not receive events for containers that are already
		// -- running when we start listening.
		containers, err := dc.ContainerList(ctx, container.ListOptions{})
		if err != nil {
			log.Warn().Err(err).Msg("unable to list containers")
		} else {
			for _, c := range containers {
				pipelineId, fnd := c.Labels["benthos_pipeline_id"]
				if !fnd {
					continue
				}
				pipelineVersion, fnd := c.Labels["benthos_pipeline_version"]
				if !fnd {
					continue
				}

				d.feedback <- &Feedback{
					Timestamp:       c.Created,
					ExecutionId:     c.ID,
					PipelineId:      pipelineId,
					PipelineVersion: pipelineVersion,
					Action:          "running",
				}
			}
		}

		for {
			select {
			case <-d.done:
				cancel()
				return
			case event := <-evts:
				pipelineId, fnd := event.Actor.Attributes["benthos_pipeline_id"]
				if !fnd {
					continue
				}
				pipelineVersion, fnd := event.Actor.Attributes["benthos_pipeline_version"]
				if !fnd {
					continue
				}

				d.feedback <- &Feedback{
					Timestamp:       event.Time,
					ExecutionId:     event.Actor.ID,
					PipelineId:      pipelineId,
					PipelineVersion: pipelineVersion,
					Action:          string(event.Action),
				}
			case err := <-errs:
				log.Warn().Err(err).Msg("docker event error")
				log.Warn().Msg("restarting docker event listener")
				evts, errs = dc.Events(ctx, options)
			}
		}
	}()

	return d, nil
}

type docker struct {
	Image string

	clientOpts []client.Opt
	dc         client.APIClient

	done     chan struct{}
	feedback chan *Feedback
}

func (d *docker) Ensure(ctx context.Context, pipelineId string, pv sdk.PipelineVersion) error {
	switch pv.Status {
	case sdk.PublishedStatus:
		exec, err := d.ensurePresent(ctx, pipelineId, pv)
		if err != nil {
			return err
		}

		return d.ensureStarted(ctx, exec.Id)

	case sdk.ArchivedStatus:
		execId, err := d.findExecutionId(ctx, pipelineId, pv.Version)
		if err != nil {
			return err
		}

		if execId == nil {
			return nil
		}

		return d.ensureAbsent(ctx, *execId)

	case sdk.PausedStatus:
		execId, err := d.findExecutionId(ctx, pipelineId, pv.Version)
		if err != nil {
			return err
		}

		if execId == nil {
			return nil
		}

		return d.ensureStopped(ctx, *execId)
	case sdk.DraftStatus:
		_, err := d.ensurePresent(ctx, pipelineId, pv)
		return err
	default:
		return fmt.Errorf("unknown pipeline status: %s", pv.Status)
	}
}

func (d *docker) Feedback() chan *Feedback {
	return d.feedback
}

func (d *docker) Close() {
	close(d.done)
}

func (d *docker) getExecution(ctx context.Context, execId string) (*Execution, error) {
	res, err := d.dc.ContainerInspect(ctx, execId)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	var state ExecutionState
	switch res.State.Status {
	case "created":
		state = PresentState
	case "restarting":
		state = StartedState
	case "running":
		state = StartedState
	case "removing":
		state = AbsentState
	case "paused":
		state = StoppedState
	case "exited":
		state = StoppedState
	case "dead":
		state = StoppedState
	default:
		state = AbsentState
	}

	return &Execution{
		res.ID,
		res.Config.Labels["benthos_pipeline_id"],
		res.Config.Labels["benthos_pipeline_version"],
		state,
	}, nil
}

func (d *docker) findExecutionId(ctx context.Context, pipelineId string, version string) (*string, error) {
	f := filters.NewArgs()
	f.Add("label", fmt.Sprintf("benthos_pipeline_id=%s", pipelineId))
	f.Add("label", fmt.Sprintf("benthos_pipeline_version=%s", version))

	containers, err := d.dc.ContainerList(ctx, container.ListOptions{Filters: f})
	if err != nil {
		return nil, fmt.Errorf("error retrieving execution: %w", err)
	}

	if len(containers) == 0 {
		return nil, nil
	}

	result := containers[0].ID
	return &result, nil
}

func (d *docker) ensurePresent(ctx context.Context, pipelineId string, pv sdk.PipelineVersion) (*Execution, error) {
	exId, err := d.findExecutionId(ctx, pipelineId, pv.Version)
	if err != nil {
		return nil, err
	}

	if exId != nil {
		ex, err := d.getExecution(ctx, *exId)
		if err != nil {
			return nil, err
		}

		// -- reached our desired state
		return ex, nil
	}

	return d.createExecution(ctx, pipelineId, pv)
}

func (d *docker) ensureStarted(ctx context.Context, execId string) error {
	ex, err := d.getExecution(ctx, execId)
	if err != nil {
		return err
	}

	if ex != nil && ex.Status == StartedState {
		return nil
	}

	return d.stopExecution(ctx, ex.Id, 10)
}

func (d *docker) ensureStopped(ctx context.Context, execId string) error {
	ex, err := d.getExecution(ctx, execId)
	if err != nil {
		return err
	}

	if ex == nil || ex.Status == StoppedState {
		return nil
	}

	return d.stopExecution(ctx, ex.Id, 10)
}

func (d *docker) ensureAbsent(ctx context.Context, execId string) error {
	ex, err := d.getExecution(ctx, execId)
	if err != nil {
		return err
	}

	if ex == nil {
		return nil
	}

	if err := d.stopExecution(ctx, ex.Id, 10); err != nil {
		return err
	}

	return d.removeExecution(ctx, ex.Id)
}
