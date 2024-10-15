package httpLib

import (
	"github.com/gc-9/gf/errors"
	"github.com/labstack/echo/v4"
	"reflect"
	"runtime"
	"strings"
)

type Route struct {
	Method      string
	Path        string
	Name        string
	HandlerFunc echo.HandlerFunc

	FuncName   string
	SourceFunc interface{} // func( [] | [context,struct] | [context] ) ( [] | [error] | [struct,error] )
	// for api doc
	InTypes  []reflect.Type
	OutTypes []reflect.Type
}

func NewRoute(method, path, name string, sourceFunc any) *Route {
	parsed, err := ParseHandler(sourceFunc)
	if err != nil {
		panic(err)
	}

	return &Route{
		Method:      strings.ToUpper(method),
		Path:        path,
		Name:        name,
		HandlerFunc: parsed.HandlerFunc,
		//
		FuncName:   parsed.FuncName,
		SourceFunc: sourceFunc,
		InTypes:    parsed.In,
		OutTypes:   parsed.Out,
	}
}

type Router interface {
	Routes() []*Route
}

var (
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
	contextType = reflect.TypeOf((*echo.Context)(nil)).Elem()

	userContextType1 = reflect.TypeOf((*RequestContext)(nil)).Elem()
)

func isContextType(t reflect.Type) bool {
	return t == contextType || t.Implements(contextType)
}

func isStructType(t reflect.Type) bool {
	if t.Kind() == reflect.Struct {
		return true
	}
	if t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct {
		return true
	}
	return false
}

func newContextValue(t reflect.Type, c echo.Context) reflect.Value {
	tp := t
	if t.Kind() == reflect.Pointer {
		tp = t.Elem()
	}
	if tp == userContextType1 {
		ctx := c.(RequestContext)
		return reflect.ValueOf(ctx)
	}

	return reflect.ValueOf(c)
}

type ParsedHandler struct {
	FuncName    string
	HandlerFunc echo.HandlerFunc
	In          []reflect.Type
	Out         []reflect.Type
}

func makeHandlerFunc(funcValue reflect.Value, inTypes []reflect.Type) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		inValue := make([]reflect.Value, len(inTypes))
		for i, inType := range inTypes {
			if isContextType(inType) {
				inValue[i] = newContextValue(inType, ctx)
			} else {
				nType := inType
				if inType.Kind() == reflect.Pointer {
					nType = inType.Elem()
				}
				rv, err := parseRequestParam(ctx, nType)
				if err != nil {
					return SendResponse(ctx, nil, err)
				}
				if inType.Kind() == reflect.Pointer {
					inValue[i] = *rv
				} else {
					inValue[i] = (*rv).Elem()
				}
			}
		}

		outValues := funcValue.Call(inValue)

		if len(outValues) == 1 {
			e := outValues[0].Convert(errorType).Interface()
			var err error
			if e != nil {
				err = e.(error)
			}
			return SendResponse(ctx, nil, err)
		} else if len(outValues) == 2 {
			data := outValues[0].Interface()
			e := outValues[1].Convert(errorType).Interface()
			var err error
			if e != nil {
				err = e.(error)
			}
			return SendResponse(ctx, data, err)
		}
		return nil
	}
}

func funcName(funcValue reflect.Value) string {
	// api/controller.(*passportController).Login-fm
	n := strings.TrimSuffix(runtime.FuncForPC(funcValue.Pointer()).Name(), "-fm")
	ns := strings.Split(n, ".")
	if len(ns) == 3 {
		return ns[1][1:len(ns[1])-1] + "." + ns[2]
	}
	return n
}

func ParseHandler(f interface{}) (*ParsedHandler, error) {
	funcValue := reflect.ValueOf(f)
	t := funcValue.Type()

	// check kind
	if funcValue.Kind() != reflect.Func {
		return nil, errors.WithStackf("%s is not a function", t)
	}

	fcName := funcName(funcValue)
	numOut := t.NumOut()
	numIn := t.NumIn()

	var outTypes []reflect.Type
	// check out count
	if numOut > 2 {
		return nil, errors.WithStackf("handler %s return count need 0~2, got %d", funcName, numOut)
	}
	// check last out type must be error
	if numOut > 0 {
		oLast := t.Out(numOut - 1)
		if oLast != errorType {
			return nil, errors.WithStackf("handler %s last return need error type. got %s", funcName, oLast)
		}

		for i := 0; i < numOut; i++ {
			outTypes = append(outTypes, t.Out(i))
		}
	}

	var inTypes []reflect.Type
	// check in count
	if numIn > 2 {
		return nil, errors.WithStackf("handler %s param count need 0~2, got %d", funcName, numIn)
	}
	// check in only 1 struct
	if numIn == 2 {
		in1 := t.In(0)
		in2 := t.In(1)
		if !isContextType(in1) && !isContextType(in2) {
			return nil, errors.WithStackf("handler %s param support 1 struct. got 2 struct", funcName)
		}
	}
	for i := 0; i < numIn; i++ {
		in := t.In(i)
		// check in type
		if !isStructType(in) && !isContextType(in) {
			return nil, errors.WithStackf("handler %s param %d need struct or %s. got %s", funcName, i+1, contextType, in)
		}
		inTypes = append(inTypes, in)
	}

	return &ParsedHandler{
		FuncName:    fcName,
		HandlerFunc: makeHandlerFunc(funcValue, inTypes),
		In:          inTypes,
		Out:         outTypes,
	}, nil
}

type paramAfterBind interface {
	AfterBind()
}

// IgnoreAutoValidate inject param disable auto validate
type ignoreAutoValidator interface {
	IgnoreAutoValidate()
}

func parseRequestParam(ctx echo.Context, paramType reflect.Type) (rv *reflect.Value, err error) {
	v := reflect.New(paramType).Interface()

	// decode
	err = ctx.Bind(v)
	if err != nil {
		if e, ok := err.(*echo.HTTPError); ok && e.Internal != nil {
			err = errors.Wrap(e.Internal, "paramError")
		}
		return
	}

	// check nil
	if p, ok := v.(paramAfterBind); ok {
		p.AfterBind()
	}

	// valid
	_, ok := v.(ignoreAutoValidator)
	if !ok {
		err = ctx.Validate(v)
		if err != nil {
			return
		}
	}

	tmp := reflect.ValueOf(v)
	rv = &tmp
	return
}
