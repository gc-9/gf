package httplib

import (
	"bytes"
	"context"
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/i18n"
	"github.com/gc-9/gf/logger"
	"github.com/gc-9/gf/util"
	"github.com/gc-9/gf/util/http_util"
	"github.com/gc-9/gf/validator"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/fx"
	"io"
	"strings"
	"time"
)

func NewServer(conf *config.Config, i18n i18n.I18n, servConf *config.Server) (*echo.Echo, error) {

	// Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// static
	for _, v := range servConf.Statics {
		e.Static(v.Path, v.Root)
	}

	// dataValidator
	dataValidator := validator.NewDataValidator()

	// context
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := ContextPoolGet(c, conf, i18n, dataValidator)
			defer ContextPoolRelease(cc)

			return next(cc)
		}
	})

	// todo
	e.HTTPErrorHandler = HandlerDefaultHTTPError(i18n)

	// logger format
	e.Logger.SetHeader(`time_rfc3339_nano ${level} ${short_file}:${line}`)

	// request log
	if servConf.DumpBody {
		noCaller := logger.NoCaller()
		const maxDumpLength = 500
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {

			return func(c echo.Context) (err error) {
				req := c.Request()
				res := c.Response()

				// Request
				reqBody := []byte{}
				if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
					if c.Request().Body != nil { // Read
						reqBody, _ = io.ReadAll(c.Request().Body)
					}
					c.Request().Body = io.NopCloser(bytes.NewBuffer(reqBody)) // Reset
				}

				// Response
				resBody := new(bytes.Buffer)
				mw := io.MultiWriter(c.Response().Writer, resBody)
				writer := &http_util.BodyDumpResponseWriter{Writer: mw, ResponseWriter: c.Response().Writer}
				c.Response().Writer = writer

				start := time.Now()
				if err = next(c); err != nil {
					c.Error(err)
				}

				reqDump := ""
				if req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH" {
					if len(reqBody) > 0 {
						if strings.HasPrefix(req.Header.Get("Content-Type"), echo.MIMEApplicationJSON) {
							reqDump = string(util.SubUtf8Bytes(reqBody, maxDumpLength))
						} else {
							reqDump, _ = http_util.DumpRequestForm(req)
						}
					}
				}

				resDump := string(util.SubUtf8Bytes(resBody.Bytes(), maxDumpLength))

				tpl := "[request] %v %v %v %v %v"
				args := []any{c.RealIP(), req.Method, req.URL, res.Status, time.Now().Sub(start)}

				if reqDump != "" {
					tpl = tpl + "\n[payload]\n%v"
					args = append(args, reqDump)
				}
				if resDump != "" {
					tpl = tpl + "\n[response]\n%v"
					args = append(args, resDump)
				}

				noCaller.Debugf(tpl, args...)
				return
			}

		})
	} else {
		noCaller := logger.NoCaller()
		e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			LogURI:      true,
			LogStatus:   true,
			LogRemoteIP: true,
			LogMethod:   true,
			LogLatency:  true,
			LogError:    true,
			LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
				if v.Error != nil {
					noCaller.Debugf("[request] %v %v %v %v %v err:%v", v.RemoteIP, v.Method, v.URI, v.Status, v.Latency, v.Error)
				} else {
					noCaller.Debugf("[request] %v %v %v %v %v", v.RemoteIP, v.Method, v.URI, v.Status, v.Latency)
				}
				return nil
			},
		}))
	}

	// limit
	//e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(10)))

	e.Use(middleware.CORS())
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize:         4 << 10, // 4 KB
		DisablePrintStack: false,
		DisableStackAll:   true,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			logger.NoCaller().Errorf("[PANIC RECOVER] %v %s\n", err, stack)

			// HandlerDefaultHTTPError() will handle this return. So return nil.
			return nil
		},
	}))

	return e, nil
}

func StartServer(lc fx.Lifecycle, conf *config.Server, srv *echo.Echo) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Logger().Debugf("Starting HTTP server at %v", conf.Addr)
			go srv.Start(conf.Addr)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}
