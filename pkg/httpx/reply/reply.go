package reply

import (
	"context"
	"net/http"

	"git.appkode.ru/pub/go/failure"
	jsoniter "github.com/json-iterator/go"

	"go-backend-example/pkg/contextx"
	"go-backend-example/pkg/errcodes"
	"go-backend-example/pkg/logx"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary //nolint:gochecknoglobals // skip

type errorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	SupportID string `json:"supportId"`
}

func (e *errorResponse) WithDefaultCode(code failure.ErrorCode) {
	if e.Code == "" {
		e.Code = code.String()
	}
}

var logger = contextx.LoggerFromContextOrDefault //nolint:gochecknoglobals

func OK(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

func Created(w http.ResponseWriter) {
	w.WriteHeader(http.StatusCreated)
}

func JSON(ctx context.Context, w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger(ctx).Error("json.Encode", logx.Error(err))
	}
}

func Error(ctx context.Context, w http.ResponseWriter, err error) {
	logger(ctx).Error("error", logx.Error(err))

	response := errorResponse{
		Code:      failure.Code(err).String(),
		Message:   failure.Description(err),
		SupportID: supportID(ctx),
	}

	switch {
	case failure.IsInvalidArgumentError(err):
		response.WithDefaultCode(errcodes.ValidationError)
		JSON(ctx, w, http.StatusBadRequest, response)
	case failure.IsNotFoundError(err):
		response.WithDefaultCode(errcodes.NotFound)
		JSON(ctx, w, http.StatusNotFound, response)
	case failure.IsUnauthorizedError(err):
		JSON(ctx, w, http.StatusUnauthorized, response)
	case failure.IsForbiddenError(err):
		response.WithDefaultCode(errcodes.Forbidden)
		JSON(ctx, w, http.StatusForbidden, response)
	case failure.IsConflictError(err):
		JSON(ctx, w, http.StatusConflict, response)
	case failure.IsUnprocessableEntityError(err):
		JSON(ctx, w, http.StatusUnprocessableEntity, response)
	default:
		response.WithDefaultCode(errcodes.InternalServerError)
		JSON(ctx, w, http.StatusInternalServerError, response)
	}
}

func supportID(ctx context.Context) string {
	traceID, err := contextx.TraceIDFromContext(ctx)
	if err != nil {
		return "unsupported"
	}

	return traceID.String()
}
