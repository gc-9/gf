package httplib

import (
	stdErrors "errors"
	"net/http"
	"strings"

	appErrors "github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/types"
	"github.com/gc-9/gf/validator"
	"github.com/labstack/echo/v4"
)

type stackTracer interface {
	StackTrace() appErrors.StackTrace
}

func SendResponse(c echo.Context, data interface{}, err error) error {
	ctx := c.(RequestContext)

	if err != nil {
		code := types.StatusCodeError
		message := ctx.I18n("error")

		switch e := err.(type) {
		case *echo.HTTPError:
			if e.Internal != nil {
				err = e.Internal
			}
		case validator.ValidationErrorsTranslations:
			message = ctx.I18n("paramError") + ":" + e.Error()
		}

		var appErr *appErrors.ErrMessage
		if stdErrors.As(err, &appErr) {
			if appErr.Code > 0 {
				code = appErr.Code
			}
			if appErr.Public {
				message = appErr.Message
			}
		}

		// log stackTracer error
		if e2, ok := err.(stackTracer); ok {
			st := e2.StackTrace()
			if len(st) > 0 {
				ctx.Log().Errorf("%s%+v", err, st)
			}
		}

		if ctx.Config().App.Env != "online" {
			debugMsg := err.Error()
			if !strings.Contains(message, debugMsg) {
				message = ctx.I18n(message) + ", debug:" + debugMsg
			}
		}

		return ctx.JSON(
			http.StatusOK,
			&types.JsonResponse{Code: code, Message: ctx.I18n(message), Data: data},
		)
	}

	if data == nil {
		return ctx.JSON(http.StatusOK, types.SuccessResponse)
	}

	return ctx.JSON(
		http.StatusOK,
		&types.JsonResponse{Code: types.StatusCodeSuccess, Message: "ok", Data: data},
	)
}
