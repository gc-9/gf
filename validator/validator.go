package validator

import (
	enLocale "github.com/go-playground/locales/en"
	zhLocale "github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
	"github.com/sleagon/chinaid"
	"reflect"
	"strings"
)

func NewDataValidator() *DataValidator {
	zh := zhLocale.New()
	en := enLocale.New()
	translator := ut.New(zh, zh, en)
	zhTrans, _ := translator.GetTranslator("zh")
	enTrans, _ := translator.GetTranslator("en")
	v := validator.New()
	_ = zhTranslations.RegisterDefaultTranslations(v, zhTrans)
	_ = enTranslations.RegisterDefaultTranslations(v, enTrans)

	// todo i18n support
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		/*tag := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		// skip if tag key says it should be ignored
		if tag == "-" {
			return ""
		}*/
		name := field.Tag.Get("comment")
		return name
	})
	v.RegisterValidation("idNumber", func(fl validator.FieldLevel) bool {
		v := chinaid.IDCard(fl.Field().String())
		return v.Valid()
	})
	v.RegisterTranslation("idNumber", zhTrans, func(ut ut.Translator) error {
		return ut.Add("idNumber", "身份证号有误", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("idNumber", fe.Field())
		return t
	})
	return &DataValidator{validate: v, translator: translator}
}

type DataValidator struct {
	translator *ut.UniversalTranslator
	validate   *validator.Validate
}

type ValidationErrorsTranslations map[string]string

func (t ValidationErrorsTranslations) Error() string {
	var tt []string
	for _, v := range t {
		tt = append(tt, v)
	}
	return strings.Join(tt, "\n")
}

func (cv *DataValidator) Validate(i interface{}, locale string) error {
	err := cv.validate.Struct(i)
	if err != nil {
		// translate all error at once
		errs := err.(validator.ValidationErrors)
		trans, _ := cv.translator.GetTranslator(locale)
		et := errs.Translate(trans)
		return ValidationErrorsTranslations(et)
	}
	return err
}
