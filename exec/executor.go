package exec

import (
	"context"
	"github.com/shono-io/nexthos/sdk"
)

type (
	Config struct {
		FromEnv bool
		Url     string
	}

	Executor interface {
		Ensure(ctx context.Context, pipelineId string, pv sdk.PipelineVersion) error
		Feedback() chan *Feedback
		Close()
	}

	Execution struct {
		Id         string
		PipelineId string
		Version    string
		Status     ExecutionState
	}

	FeedbackAction string

	Feedback struct {
		Timestamp       int64
		PipelineId      string
		PipelineVersion string
		ExecutionId     string
		Action          string
		Data            any
	}

	ExecutionState string

	Job struct {
		ID            string
		ProcessorPath string
	}
)

const (
	PresentState ExecutionState = "present"
	StartedState ExecutionState = "started"
	StoppedState ExecutionState = "stopped"
	AbsentState  ExecutionState = "absent"
)

const (
	StartedFeedbackAction FeedbackAction = "started"
)
