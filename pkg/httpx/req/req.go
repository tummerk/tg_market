package req

import (
	"fmt"
	"net/http"

	"git.appkode.ru/pub/go/failure"
	"github.com/go-playground/validator/v10"
	jsoniter "github.com/json-iterator/go"

	"go-backend-example/pkg/errcodes"
)

var (
	json     = jsoniter.ConfigCompatibleWithStandardLibrary         //nolint:gochecknoglobals // skip
	validate = validator.New(validator.WithRequiredStructEnabled()) //nolint:gochecknoglobals // skip
)

func Read(r *http.Request, dest any) error {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		return failure.NewInvalidArgumentError(
			fmt.Errorf("json.Decode: %w", err).Error(),
			failure.WithCode(errcodes.ValidationError),
			failure.WithDescription("Invalid JSON"),
		)
	}

	if err := validate.StructCtx(r.Context(), dest); err != nil {
		return failure.NewInvalidArgumentError(
			"validation error",
			failure.WithCode(errcodes.ValidationError),
			failure.WithDescription(err.Error()),
		)
	}

	return nil
}
