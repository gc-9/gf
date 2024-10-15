package i18n

type I18n interface {
	T(lang string, key string) string
}

type I18nMap map[string]*Lang

func (t *I18nMap) T(lang string, key string) string {
	v, ok := (*t)[key]
	if !ok {
		return key
	}
	return v.T(lang)
}

type Lang struct {
	EN string
	ZH string
}

func (t *Lang) T(lang string) string {
	if lang == "en" {
		return t.EN
	}
	return t.ZH
}
