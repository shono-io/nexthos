package repo

import (
	"context"
	"github.com/shono-io/nexthos/sdk"
)

type (
	Config struct {
		KeyValueBucket    string
		ObjectStoreBucket string
		Prefix            string
	}

	Repository interface {
		Watch(ctx context.Context)
		Updates() <-chan *Change
		Close() error
	}

	Operation string
	Change    struct {
		Operation  Operation
		PipelineId string
		Version    sdk.PipelineVersion
		Revision   uint64
	}
)

var (
	PutOperation    Operation = "put"
	DeleteOperation Operation = "del"
)
