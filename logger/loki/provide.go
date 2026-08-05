package loki

import (
	"context"

	"github.com/gc-9/gf/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ProvideClient constructs the single client shared by access logs and the
// regular Zap core.
func ProvideClient(lc fx.Lifecycle, conf *config.Config) (*Client, error) {
	var lokiConf Config
	if err := conf.Get("loki", &lokiConf); err != nil {
		return nil, err
	}
	client, err := NewClient(&lokiConf)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return client.Close(ctx)
		},
	})
	return client, nil
}

// ProvideCore creates the regular application-log Loki core. logger.InitLogger
// keeps this concrete core out of its separate request logger.
func ProvideCore(client *Client, conf *config.Config, levelFilter func(zapcore.Level) bool) zapcore.Core {
	labels := Labels{
		"app":    conf.App.Name,
		"env":    conf.App.Env,
		"source": "log",
	}
	return NewCore(client, labels, zap.LevelEnablerFunc(levelFilter))
}
