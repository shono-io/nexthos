package exec

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/shono-io/nexthos/sdk"
)

func (d *docker) stopExecution(ctx context.Context, execId string, timeout int) error {
	if err := d.dc.ContainerStop(ctx, execId, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("unable to stop container: %w", err)
	}

	return nil
}

func (d *docker) removeExecution(ctx context.Context, execId string) error {
	if err := d.dc.ContainerRemove(ctx, execId, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("unable to remove container: %w", err)
	}

	return nil
}

func (d *docker) createExecution(ctx context.Context, pipelineId string, pv sdk.PipelineVersion) (*Execution, error) {
	popts := image.PullOptions{}
	out, err := d.dc.ImagePull(ctx, d.Image, popts)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	// ideally we would log this output
	//io.Copy(log.Debug()., out)

	containerName := fmt.Sprintf("%s-%s", pipelineId, pv.Version)

	resp, err := d.dc.ContainerCreate(ctx, d.toDockerContainerConfig(pipelineId, pv), nil, nil, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("unable to create execution: %w", err)
	}

	return &Execution{
		Id:         resp.ID,
		PipelineId: pipelineId,
		Version:    pv.Version,
		Status:     PresentState,
	}, nil
}

func (d *docker) startExecution(ctx context.Context, execId string) error {
	if err := d.dc.ContainerStart(ctx, execId, container.StartOptions{}); err != nil {
		return fmt.Errorf("unable to start execution: %w", err)
	}

	return nil
}

func (d *docker) toDockerContainerConfig(pipelineId string, pv sdk.PipelineVersion) *container.Config {
	return &container.Config{
		Image: d.Image,
		Labels: map[string]string{
			"benthos_pipeline_id":      pipelineId,
			"benthos_pipeline_version": pv.Version,
		},
	}
}
