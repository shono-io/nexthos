package sdk

type PipelineStatus string

var (
	DraftStatus     PipelineStatus = "draft"
	PublishedStatus PipelineStatus = "published"
	PausedStatus    PipelineStatus = "paused"
	ArchivedStatus  PipelineStatus = "archived"
)

type Pipeline struct {
	PipelineHeader
	PipelineVersion
}

type PipelineHeader struct {
	Key         string
	Name        string
	Description string
}

type PipelineVersion struct {
	Version      string
	ContentKey   string
	ArtifactKeys []string
	Status       PipelineStatus
}
