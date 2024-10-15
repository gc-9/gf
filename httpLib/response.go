package httpLib

import (
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/logger"
	"github.com/gc-9/gf/types"
	"github.com/gc-9/gf/validator"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func SendResponse(c echo.Context, data interface{}, err error) error {
	ctx := c.(RequestContext)

	if err != nil {
		code := types.StatusCodeError

		var humanMsg = ""

		switch e := err.(type) {
		case *echo.HTTPError:
			if e.Internal != nil {
				err = e.Internal
			}
		case validator.ValidationErrorsTranslations:
			humanMsg = ctx.I18n("paramError") + ":" + e.Error()
		}

		switch e := err.(type) {
		case *errors.ErrMessage:
			if e.Code > 0 {
				code = e.Code
			}
			humanMsg = e.HumanMsg
		}

		// log stackTracer error
		if e2, ok := err.(stackTracer); ok {
			st := e2.StackTrace()
			if st != nil {
				// stacktrace humanMsg always be error
				humanMsg = "error"
				logger.NoCaller().Errorf("%s%+v", err, st[0:1])
			}
		}

		if humanMsg == "" {
			humanMsg = "error"
		}

		msg := humanMsg
		if ctx.Config().App.Env != "online" {
			debugMsg := err.Error()
			if !strings.Contains(humanMsg, debugMsg) {
				msg = ctx.I18n(humanMsg) + ", debug:" + debugMsg
			}
		}

		return ctx.JSON(
			http.StatusOK,
			&types.JsonResponse{Code: code, Message: ctx.I18n(msg), Data: data},
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
