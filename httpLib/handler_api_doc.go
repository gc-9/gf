package httpLib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gc-9/gf/config"
	"github.com/labstack/echo/v4"
	"html/template"
	"reflect"
	"regexp"
	"strings"
)

var docTemplate = `
<!doctype html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>API DOC</title>
</head>
<body>
<style>
body { font-size: 14px; }
pre { background: #f0f0f0;  padding: 10px;  border-radius: 5px; max-width: 800px; margin: 5px 0 0; }
h4 { margin: 10px 0 5px; }
details pre { background: none; margin: 0; }
.menu { position: fixed; right: 10px; top: 10px; border: 1px solid #ccc; padding: 10px 20px 10px 30px; min-width: 160px; max-height: 50vh; overflow-y: auto; }
.menu a { color: inherit; text-decoration: none }
.menu a:hover { text-decoration:underline; }
</style>

<ol class="menu">
{{range $i, $api := .Apis}}<li><a href="#api-{{$i}}">{{ $api.Name }}</a></li>{{end}}
</ol>

<div>@baseUrl = {{ .BaseUrl }}</div>
<br>
{{range $i, $api := .Apis}}<h4 id="api-{{$i}}">### {{ $api.Name }}</h4>
{{if ne $api.In nil }}
<details>
  <summary>//# payload</summary>
  <pre>{{docParam $api.In "// "}}</pre>
</details>
{{end}}{{if ne $api.Out nil }}
<details>
  <summary>//# response</summary>
  <pre>{{docParam $api.Out "// "}}</pre>
</details>
{{end}}

<pre>
{{$api.Method}} {{"{{ baseUrl }}"}}{{$api.Path}}{{if ne $api.In nil }}
Content-Type: application/json{{end}}
{{if ne $api.In nil }}
{{$api.In.Example}}{{end}}
</pre>
{{end}}

</body>
`

func HandlerApiDoc(cfg *config.Server, routes []*Route) echo.HandlerFunc {
	baseUrl := cfg.Url + cfg.Prefix

	return func(ctx echo.Context) error {

		tpl := template.New("apiDoc")
		tpl.Funcs(template.FuncMap{
			"docParam": func(param *ApiDocParam, prefix string) string {
				r := regexp.MustCompile("(?m)^")
				str := docParam(param)
				return r.ReplaceAllString(str, prefix)
			},
		})

		tmpl, err := tpl.Parse(docTemplate)
		if err != nil {
			return ctx.HTML(200, err.Error())
		}

		var apis []*ApiDoc
		for _, r := range routes {
			fcName := r.Name
			if fcName == "" {
				fcName = r.FuncName
			}

			api := &ApiDoc{
				Name:   fcName,
				Method: r.Method,
				Path:   r.Path,
				In:     parseRealParams(r.InTypes),
				Out:    parseRealParams(r.OutTypes),
			}

			apis = append(apis, api)
		}

		buffer := bytes.NewBufferString("")
		err = tmpl.Execute(buffer, map[string]interface{}{
			"Apis":    apis,
			"BaseUrl": baseUrl,
		})

		if err != nil {
			return ctx.HTML(200, err.Error())

		}

		return ctx.HTML(200, buffer.String())
	}
}

type ApiDoc struct {
	Name   string
	Method string
	Path   string
	In     *ApiDocParam
	Out    *ApiDocParam
}

type ApiDocParam struct {
	Type    reflect.Type
	Example string
	Fields  []*ApiDocParamField
}

type ApiDocParamField struct {
	Name     string
	Type     reflect.Type
	Comment  string
	Validate string
	Example  string
}

func docParam(param *ApiDocParam) string {
	if param == nil {
		return "unknown"
	}

	var fields []string

	l := len(param.Fields)
	for i, f := range param.Fields {
		dot := ","
		if i+1 == l {
			dot = ""
		}
		comments := []string{f.Type.String()}
		if f.Comment != "" {
			comments = append(comments, f.Comment)
		}
		fieldStr := fmt.Sprintf(`"%s": %s%s // %s`, f.Name, f.Example, dot, strings.Join(comments, ", "))
		fields = append(fields, fmt.Sprintf("  %s", fieldStr))
	}
	return "{\n" + strings.Join(fields, "\n") + "\n}"
}

var cacheApiParam = map[reflect.Type]*ApiDocParam{}

func parseParam(t reflect.Type) *ApiDocParam {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if apiParam, ok := cacheApiParam[t]; ok {
		return apiParam
	}

	var fields []*ApiDocParamField

	n := t.NumField()
	for i := 0; i < n; i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		// tag and comments
		tagComment := sf.Tag.Get("comment")
		tagValidate := sf.Tag.Get("validate")
		tagName := sf.Tag.Get("json")

		// ignore field
		if tagName == "-" {
			continue
		}

		tagName = strings.Split(tagName, ",")[0]
		if tagName == "" {
			tagName = sf.Name
			tagName = strings.ToLower(tagName[:1]) + tagName[1:]
		}

		// defaultValue
		var defaultValue string
		var v reflect.Value
		if sf.Type.Kind() == reflect.Pointer {
			v = reflect.New(sf.Type.Elem()).Elem()
		} else {
			v = reflect.New(sf.Type).Elem()
		}
		defaultValueBuf, _ := json.Marshal(v.Interface())
		defaultValue = string(defaultValueBuf)

		f := &ApiDocParamField{
			Name:     tagName,
			Type:     sf.Type,
			Comment:  tagComment,
			Validate: tagValidate,
			Example:  defaultValue,
		}

		fields = append(fields, f)
	}

	defaultValueBuf, _ := json.MarshalIndent(reflect.New(t).Elem().Interface(), "", "  ")
	apiParam := &ApiDocParam{
		Type:    t,
		Example: string(defaultValueBuf),
		Fields:  fields,
	}

	cacheApiParam[t] = apiParam
	return apiParam
}

func parseRealParams(types []reflect.Type) *ApiDocParam {
	for _, t := range types {
		if !isStructType(t) {
			continue
		}
		return parseParam(t)
	}
	return nil
}
