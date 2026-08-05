package httplib

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/i18n"
	"github.com/gc-9/gf/logger"
	"github.com/gc-9/gf/logger/loki"
	"github.com/gc-9/gf/util"
	"github.com/gc-9/gf/util/http_util"
	"github.com/gc-9/gf/validator"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/fx"
)

type newServerParams struct {
	fx.In

	Conf       *config.Config
	I18n       i18n.I18n
	Server     *config.Server
	LokiClient *loki.Client `optional:"true"`
}

// NewServer is the Fx constructor. LokiClient is an optional dependency, so
// the server works with or without loki.ProvideClient.
func NewServer(params newServerParams) (*echo.Echo, error) {
	return newServer(params.Conf, params.I18n, params.Server, params.LokiClient)
}

func newServer(conf *config.Config, i18n i18n.I18n, servConf *config.Server, lokiClient *loki.Client) (*echo.Echo, error) {
	if err := servConf.RequestLog.Compile(); err != nil {
		return nil, err
	}

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

	requestLabels := loki.Labels{
		"app":     conf.App.Name,
		"service": servConf.Name,
		"env":     conf.App.Env,
		"source":  "request",
	}

	// request log
	if servConf.RequestLog.DumpBody {
		requestLogger := logger.RequestNoCaller()
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

				resDump := ""
				resContentType := res.Header().Get("Content-Type")
				if strings.HasPrefix(resContentType, echo.MIMEApplicationJSON) {
					resDump = string(util.SubUtf8Bytes(resBody.Bytes(), maxDumpLength))
				} else {
					resDump = resContentType + "\n--no dump-- "
				}

				latency := time.Since(start)
				tpl := "[request] %v %v %v %v %v"
				args := []any{c.RealIP(), req.Method, req.URL, res.Status, latency}

				if reqDump != "" {
					tpl = tpl + "\n[payload]\n%v"
					args = append(args, reqDump)
				}
				if resDump != "" {
					tpl = tpl + "\n[response]\n%v"
					args = append(args, resDump)
				}

				if servConf.RequestLog.ShouldLog(req.URL.Path, res.Status, err) {
					pushRequestLog(lokiClient, requestLabels, req, res.Status, c.RealIP(), latency, err, reqDump, resDump)
					requestLogger.Debugf(tpl, args...)
				}
				return
			}

		})
	} else {
		requestLogger := logger.RequestNoCaller()
		e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			LogURI:      true,
			LogStatus:   true,
			LogRemoteIP: true,
			LogMethod:   true,
			LogLatency:  true,
			LogError:    true,
			LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
				request := c.Request()
				if servConf.RequestLog.ShouldLog(request.URL.Path, v.Status, v.Error) {
					pushRequestLog(lokiClient, requestLabels, request, v.Status, v.RemoteIP, v.Latency, v.Error, "", "")
					if v.Error != nil {
						requestLogger.Debugf("[request] %v %v %v %v %v err:%v", v.RemoteIP, v.Method, v.URI, v.Status, v.Latency, v.Error)
					} else {
						requestLogger.Debugf("[request] %v %v %v %v %v", v.RemoteIP, v.Method, v.URI, v.Status, v.Latency)
					}
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
			return err
		},
	}))

	return e, nil
}

func pushRequestLog(client *loki.Client, labels loki.Labels, req *http.Request, status int, remoteIP string, latency time.Duration, requestErr error, requestBody, responseBody string) {
	if client == nil || !client.Enabled() {
		return
	}

	level := "debug"
	if requestErr != nil || status >= http.StatusInternalServerError {
		level = "error"
	}
	line := map[string]any{
		"level":      level,
		"method":     req.Method,
		"path":       req.URL.Path,
		"status":     status,
		"latency_ms": float64(latency) / float64(time.Millisecond),
		"remote_ip":  remoteIP,
	}
	if requestErr != nil {
		line["error"] = requestErr.Error()
	}
	if requestBody != "" {
		line["request_body"] = requestBody
	}
	if responseBody != "" {
		line["response_body"] = responseBody
	}
	encoded, err := json.Marshal(line)
	if err == nil {
		client.Push(labels, time.Now(), encoded)
	}
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
