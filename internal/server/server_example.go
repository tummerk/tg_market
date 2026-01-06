package server

import (
	"context"
	"fmt"
	"net/http"

	"git.appkode.ru/pub/go/failure"

	"go-backend-example/internal/domain/entity"
	"go-backend-example/internal/domain/value"
	"go-backend-example/pkg/errcodes"
	"go-backend-example/pkg/httpx/reply"
	"go-backend-example/pkg/httpx/req"
	"go-backend-example/pkg/rest"
)

type exampleService interface {
	GetByID(context.Context, value.ExampleID) (entity.Example, error)
	Save(context.Context, entity.Example) error
}

type ExampleServer struct {
	exampleService exampleService
}

func NewExampleServer(exampleService exampleService) ExampleServer {
	return ExampleServer{
		exampleService: exampleService,
	}
}

func (s ExampleServer) getV1Example(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	id, err := value.ParseExampleID(r.PathValue("id"))
	if err != nil {
		return fmt.Errorf("value.ParseExampleID: %w", err)
	}

	example, err := s.exampleService.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("exampleService.GetByID: %w", err)
	}

	reply.JSON(ctx, w, http.StatusOK, newRESTExample(example))

	return nil
}

func (s ExampleServer) postV1Example(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	var request rest.Example

	if err := req.Read(r, &request); err != nil {
		return fmt.Errorf("req.Read: %w", err)
	}

	example, err := newDomainExample(request)
	if err != nil {
		return failure.NewInvalidArgumentErrorFromError(
			fmt.Errorf("newDomainExample: %w", err),
			failure.WithCode(errcodes.InvalidExample),
		)
	}

	if err = s.exampleService.Save(ctx, example); err != nil {
		return fmt.Errorf("exampleService.Save: %w", err)
	}

	reply.OK(w)

	return nil
}
