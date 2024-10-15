package httpLib

import (
	"github.com/gc-9/gf/i18n"
	"github.com/gc-9/gf/types"
	"github.com/labstack/echo/v4"
	"net/http"
)

func HandlerDefaultHTTPError(i18n i18n.I18n) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		httpStatus := http.StatusOK
		code := types.StatusCodeError

		// may be nil
		if err == nil {
			_ = c.JSON(
				httpStatus,
				&types.JsonResponse{Code: code, Message: i18n.T(GetLocale(c), "error")},
			)
			return
		}

		msg := ""
		he, ok := err.(*echo.HTTPError)
		if ok {
			if he.Internal != nil {
				if herr, ok := he.Internal.(*echo.HTTPError); ok {
					he = herr
				}
			}
			if he.Code == http.StatusNotFound {
				httpStatus = http.StatusNotFound
				code = types.StatusCodeNotFound
				msg = he.Error()
			}
		} else {
			msg = i18n.T(GetLocale(c), "error")
		}

		_ = c.JSON(
			httpStatus,
			&types.JsonResponse{Code: code, Message: msg},
		)
	}
}
