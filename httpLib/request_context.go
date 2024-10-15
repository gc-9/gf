package httpLib

import (
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/i18n"
	"github.com/gc-9/gf/validator"
	"github.com/labstack/echo/v4"
	"golang.org/x/text/language"
	"sync"
)

var poolContext = sync.Pool{}

func init() {
	poolContext.New = func() any {
		return &CommonRequestContext{}
	}
}

type RequestContext interface {
	echo.Context
	GetLocale() string
	I18n(key string) string
	Config() *config.Config

	AuthUser() any
}

func ContextPoolGet(c echo.Context, conf *config.Config, i18b i18n.I18n, validator *validator.DataValidator) RequestContext {
	ctx := poolContext.Get().(*CommonRequestContext)
	ctx.Context = c
	ctx.i18n = i18b
	ctx.config = conf
	ctx.validator = validator
	return ctx
}

func ContextPoolRelease(cc RequestContext) {
	// reset
	ctx := cc.(*CommonRequestContext)
	poolContext.Put(ctx)
}

type CommonRequestContext struct {
	echo.Context
	i18n      i18n.I18n
	config    *config.Config
	validator *validator.DataValidator
}

func (c *CommonRequestContext) Config() *config.Config {
	return c.config
}

// GetLocale language subtags @see BCP 47 https://en.wikipedia.org/wiki/IETF_language_tag
func (c *CommonRequestContext) GetLocale() string {
	return GetLocale(c)
}

func GetLocale(c echo.Context) string {
	l := c.Get("locale")
	if l != nil {
		return l.(string)
	}

	var matcher = language.NewMatcher([]language.Tag{
		language.Chinese, // The first language is used as fallback.
		language.English,
	})
	accept := c.Request().Header.Get("Accept-Language")
	tag, _ := language.MatchStrings(matcher, accept)
	b, _ := tag.Base()
	locale := b.String()

	c.Set("locale", locale)
	return locale
}

func (c *CommonRequestContext) AuthUser() any {
	return c.Get("authUser")
}

//func (c *CommonRequestContext) AuthUser() *types.User {
//	if v := c.Get("user"); v != nil {
//		return v.(*types.User)
//	}
//	return nil
//}

// Validate replace validate func. add locale support
func (c *CommonRequestContext) Validate(i interface{}) error {
	return c.validator.Validate(i, c.GetLocale())
}

func (c *CommonRequestContext) I18n(key string) string {
	return c.i18n.T(c.GetLocale(), key)
}
