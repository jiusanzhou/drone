package core

import "context"

type (
	// Pipeline represents a set of stages
	Pipeline struct {
		ID          int64  `json:"id"`
		UID         string `json:"uid"`
		UserID      int64  `json:"user_id"`
		Name        string `json:"name"`
		Visibility  string `json:"visibility"`
		Created     int64  `json:"created"`
		Updated     int64  `json:"updated"`
		Version     int64  `json:"version"`
		Data        []byte `json:"data"`
	}

	
	// PipelineStore defines operations for working with pipelines.
	PipelineStore interface {

		// List returns a pipeline list from the datastore.
		List(context.Context, int64) ([]*Pipeline, error)

		// Find returns a pipeline from the datastore.
		Find(context.Context, int64) (*Pipeline, error)

		// FindName returns a named pipeline from the datastore.
		FindName(context.Context, string, string) (*Pipeline, error)

		// Create persists a new pipeline in the datastore.
		Create(context.Context, *Pipeline) error

		// Update persists pipeline changes to the datastore.
		Update(context.Context, *Pipeline) error

		// Delete deletes a pipeline from the datastore.
		Delete(context.Context, *Pipeline) error
	}
)



