package types

import (
	"encoding/json"
	"fmt"
	"github.com/gc-9/gf/errors"
	"github.com/samber/lo"
	"reflect"
	"strings"
	"time"
	"xorm.io/builder"
	"xorm.io/xorm"
)

var defaultPageSize = 20

type ParamPager struct {
	PageSize int `json:"pageSize"`
	PageNum  int `json:"pageNum"`
	Offset   int `json:"-"`
}

func (t *ParamPager) FillDefaultValue() {
	if t.PageNum <= 0 {
		t.PageNum = 1
	}
	if t.PageSize <= 0 {
		t.PageSize = defaultPageSize
	}
	t.Offset = (t.PageNum - 1) * t.PageSize
}

type ParamPageQuery struct {
	Filters     Filters `json:"filters"` //never be nil
	*ParamPager         //never be nil
}

func (p *ParamPageQuery) AfterBind() {
	if p.Filters == nil {
		p.Filters = Filters{}
	}
	if p.ParamPager == nil {
		p.ParamPager = &ParamPager{
			PageSize: defaultPageSize,
			PageNum:  0,
			Offset:   0,
		}
	}
}

func (p *ParamPageQuery) QueryOption() func(session *xorm.Session) {
	return func(session *xorm.Session) {
		session.Limit(p.PageSize, p.Offset)
		p.Filters.QueryOption()(session)
	}
}

type paramPageQuery ParamPageQuery

func (p *ParamPageQuery) UnmarshalJSON(buf []byte) error {
	var cp paramPageQuery
	err := json.Unmarshal(buf, &cp)
	if err != nil {
		return err
	}
	*p = ParamPageQuery(cp)
	if p.Filters == nil {
		p.Filters = make(Filters)
	}
	if p.ParamPager == nil {
		p.ParamPager = &ParamPager{}
	}
	p.FillDefaultValue()
	return nil
}

type Filters map[string]interface{}

func (f *Filters) QueryOption() func(session *xorm.Session) {
	return func(session *xorm.Session) {
		for k, f := range *f {
			session.Where(builder.Eq{k: f})
		}
	}
}

func (f *Filters) GetTimeRange(key string) []string {
	if v, ok := (*f)[key]; ok {
		var t1str, t2str string
		switch val := v.(type) {
		case []string:
			if len(val) == 2 {
				t1str, t2str = val[0], val[1]
			}
		case []interface{}:
			if len(val) == 2 {
				t1str, _ = val[0].(string)
				t2str, _ = val[1].(string)
			}
		}
		if t1str == "" || t2str == "" {
			return nil
		}

		t1, err1 := time.Parse("2006-01-02", t1str)
		t2, err2 := time.Parse("2006-01-02", t2str)
		if err1 == nil && err2 == nil {
			return []string{t1.Format("2006-01-02"), t2.AddDate(0, 0, 1).Format("2006-01-02")}
		}
	}
	return nil
}

func (f *Filters) ValueString(k string) string {
	v, ok := (*f)[k]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func (f *Filters) Allows(allows []string) {
	*f = lo.PickBy(*f, func(k string, v interface{}) bool {
		return lo.Contains(allows, k)
	})
}

func (f *Filters) UnmarshalJSON(buf []byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(buf, &m)
	if err != nil {
		return err
	}
	filters := map[string]interface{}{}
	for k, v := range m {
		switch item := v.(type) {
		case string:
			item = strings.TrimSpace(item)
			if item != "" {
				filters[k] = item
			}
		case bool, int, int64, float32, float64:
			filters[k] = item
		case []interface{}:
			if len(item) > 0 {
				filters[k] = item
			}
		default:
			return errors.Errorf("types.Filters[%s] value nonsupport type %s", k, reflect.TypeOf(item))
		}
	}

	*f = filters
	return nil
}
