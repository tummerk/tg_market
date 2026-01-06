package server

import (
	"fmt"

	"go-backend-example/internal/domain/entity"
	"go-backend-example/internal/domain/value"
	"go-backend-example/pkg/rest"
)

func newRESTExample(example entity.Example) rest.Example {
	return rest.Example{
		ID:   example.ID.String(),
		Name: example.Name.String(),
	}
}

func newDomainExample(example rest.Example) (entity.Example, error) {
	id, err := value.ParseExampleID(example.ID)
	if err != nil {
		return entity.Example{}, fmt.Errorf("value.ParseExampleID: %w", err)
	}

	return entity.Example{
		ID:   id,
		Name: value.ExampleName(example.Name),
	}, nil
}
