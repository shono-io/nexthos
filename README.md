# Benthos Nats Execution

Nats Jetstream provides the ideal constructs to build a scalable, reliable, and high-performance platform for event 
processing. Benthos on the other hand is a powerful tool for processing data streams. This project aims to combine the
two to provide a powerful and flexible platform for processing data streams.

## How It Works
The definition of a pipeline is stored both in the Jetstream KV store and a Jetstream Object Store. While the KV entry
describes the general details of the pipeline (id, name, description, status, etc), the Object Store contains the actual
benthos configuration.

This application will react to changes in the KV store and instruct an `executor` to take action. This executor might
be anything. The current implementation is based on docker, but nothing prevents us from using NEX or even Kubernetes 
as an executor.

When the application starts, it will start watching the Jetstream KV store for changes and acts according to the
behavior explained below.

#### PUT: a pipeline is added or updated
To begin with, we need to know if the change actually matters. We do this by looking at the `status` field which can
have one of the following values:

- `draft`: the pipeline is not yet ready to be executed
- `published`: the pipeline is ready to be executed
- `paused`: the pipeline is paused

If a pipeline is in the `published` status, the executor will make sure the pipeline is running and the benthos 
configuration used corresponds to the one defined in the `version` field.

If a pipeline is not in the `published` status, the executor will make sure the pipeline is stopped.

#### DEL: a pipeline is removed
When a pipeline definition is removed from the KV store, the executor will make sure the pipeline is also destroyed and
lingering resources are cleaned up.

### The Executor
We made an abstraction of the component that will actually execute the pipelines. This component can take many forms,
from an in process co-routine being spawned, to docker containers being started or even kubernetes jobs being created.
For convenience, we will call what is spawned by the executor an **execution**.

The current implementation is based on docker, which means an execution corresponds to a docker container. The image
used for the container contains logic to retrieve the actual benthos definition from the nats object store and start
the benthos stream.

Publishing the pipeline will make sure an execution is running, creating one if none exists. Pausing a pipeline will
merely stop the container associated with the execution. Removing the pipeline or moving the status of the pipeline 
to `draft` will make sure any traces of the container are removed.