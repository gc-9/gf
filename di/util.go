package di

import (
	"github.com/gc-9/gf/httpLib"
	"go.uber.org/fx"
)

const routerTags = `group:"routers"`

func AsRouter(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(httpLib.Router)),
		fx.ResultTags(routerTags),
	)
}

func ProvideServices(services []any) fx.Option {
	return fx.Provide(
		services...,
	)
}

func ProvideRouters(routers []any) fx.Option {
	rs := make([]any, len(routers))
	for i, r := range routers {
		rs[i] = AsRouter(r)
	}
	return fx.Provide(
		rs...,
	)
}

func InvokeRegisterRouters(fun any) fx.Option {
	return fx.Invoke(fx.Annotate(fun, fx.ParamTags(routerTags)))
}
