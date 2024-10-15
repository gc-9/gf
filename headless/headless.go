package headless

import (
	"github.com/gc-9/gf/errors"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"sync"
	"time"
)

var browser *rod.Browser
var lock sync.Mutex
var lastErr error

func bootBrowser() (err error) {
	defer func() {
		if err1 := recover(); err1 != nil {
			err = errors.Errorf("recover painc:%v", err1)
		}
	}()
	lock.Lock()
	defer lock.Unlock()

	if lastErr != nil || browser != nil {
		return lastErr
	}
	u := launcher.New().
		UserDataDir("./tmp/user-data").
		Headless(true).
		MustLaunch()
	browser = rod.New().ControlURL(u)
	lastErr = browser.Connect()
	return lastErr
}

func GetDocument(url string, timeout time.Duration) (html string, err error) {
	return GetSome[string](url, timeout, func(page *rod.Page) (string, error) {
		return page.HTML()
	})
}

func GetSome[T comparable](url string, timeout time.Duration, funk func(page *rod.Page) (T, error)) (value T, err error) {
	defer func() {
		if err1 := recover(); err1 != nil {
			err = errors.Errorf("recover painc:%v", err1)
		}
	}()

	err = bootBrowser()
	if err != nil {
		return
	}

	//defer page.MustClose()
	page := browser.MustPage("")
	defer page.MustClose()

	router := page.HijackRequests()
	router.MustAdd("*", func(ctx *rod.Hijack) {
		if ctx.Request.Type() != proto.NetworkResourceTypeDocument {
			ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
			return
		}
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})
	go router.Run()

	if timeout != 0 {
		page = page.Timeout(timeout)
	}
	page.MustNavigate(url).MustWaitLoad()
	return funk(page)
}
