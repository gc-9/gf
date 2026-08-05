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

// ProvideCore creates a Loki Zap core using the complete label set supplied
// by the caller. logger.InitLogger keeps this concrete core out of its
// separate request logger.
//
// Example:
//
//	labels := loki.Labels{
//		"app":     conf.App.Name,
//		"env":     conf.App.Env,
//		"source":  "log",
//	}
//	core := loki.ProvideCore(client, labels, func(level zapcore.Level) bool {
//		return level >= zapcore.InfoLevel
//	})
func ProvideCore(client *Client, labels Labels, levelFilter func(zapcore.Level) bool) zapcore.Core {
	return NewCore(client, labels, zap.LevelEnablerFunc(levelFilter))
}
