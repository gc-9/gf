package rpc

import (
	"bytes"
	"encoding/gob"
	"github.com/gc-9/gf/errors"
	"net/http"
	"reflect"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func parseMethod(funcValue reflect.Value) *reflectHandler {
	funcType := funcValue.Type()

	inTypes := make([]reflect.Type, funcType.NumIn())
	for i := 0; i < funcType.NumIn(); i++ {
		inTypes[i] = funcType.In(i)
	}

	outTypes := make([]reflect.Type, funcType.NumOut())
	for i := 0; i < funcType.NumOut(); i++ {
		outTypes[i] = funcType.Out(i)
	}

	return &reflectHandler{
		caller:   funcValue,
		inTypes:  inTypes,
		outTypes: outTypes,
	}
}

type reflectHandler struct {
	caller   reflect.Value
	inTypes  []reflect.Type
	outTypes []reflect.Type
}

func NewRemoteInject(remoteUrl string) *RemoteInject {
	return &RemoteInject{remoteUrl: remoteUrl}
}

type RemoteInject struct {
	remoteUrl string
	handlers  map[string]*reflectHandler
}

func (t *RemoteInject) Inject(parent any) {
	// self method names
	methodMap := make(map[string]struct{})
	selfType := reflect.TypeOf(t)

	for i := 0; i < selfType.NumMethod(); i++ {
		method := selfType.Method(i)
		if method.PkgPath == "" {
			methodMap[method.Name] = struct{}{}
		}
	}

	tValue := reflect.ValueOf(parent)
	tType := tValue.Type()

	handlers := make(map[string]*reflectHandler)
	for i := 0; i < tType.NumMethod(); i++ {
		method := tType.Method(i)
		// Skip unexported methods
		if method.PkgPath != "" {
			continue
		}
		// skip self methods
		if _, ok := methodMap[method.Name]; ok {
			continue
		}
		funcNoReceiver := tValue.Method(i)
		handlers[method.Name] = parseMethod(funcNoReceiver)
	}
	t.handlers = handlers
}

func (t *RemoteInject) Call(method string, argsRaw []any) ([]any, error) {
	handler, _ := t.handlers[method]

	if len(argsRaw) != len(handler.inTypes) {
		return nil, errors.New("param error")
	}

	var args [][]byte
	for _, arg := range argsRaw {
		buf, err := gobEncode(arg)
		if err != nil {
			return nil, errors.Wrap(err, "args encode error")
		}
		args = append(args, buf)
	}

	buf, err := gobEncode(args)
	if err != nil {
		return nil, errors.Wrap(err, "args encode error")
	}

	resp, err := http.DefaultClient.Post(t.remoteUrl+"?method="+method, "application/json", bytes.NewBuffer(buf))
	if err != nil {
		return nil, errors.Wrap(err, "http post error")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("http status code %d", resp.StatusCode)
	}

	var outsString [][]byte
	decoder := gob.NewDecoder(resp.Body)
	err = decoder.Decode(&outsString)
	if err != nil {
		return nil, errors.Wrap(err, "decode response error")
	}

	if len(outsString) != len(handler.outTypes) {
		return nil, errors.New("response length error")
	}

	// outs -> handler.outTypes
	outsValue := make([]any, len(handler.outTypes))
	for i, outType := range handler.outTypes {
		if outsString[i] == nil {
			outsValue[i] = nil
			continue
		}
		if outType.Implements(errorType) {
			if len(outsString[i]) > 0 {
				outsValue[i] = errors.New(string(outsString[i]))
			} else {
				outsValue[i] = nil
			}
			continue
		}

		isPtr := outType.Kind() == reflect.Ptr
		if isPtr {
			outType = outType.Elem()
		}
		val := reflect.New(outType).Interface()
		err := gobDecode(outsString[i], val)
		if err != nil {
			return nil, errors.Wrap(err, "decode response error")
		}
		outsValue[i] = val
		if !isPtr {
			outsValue[i] = reflect.ValueOf(val).Elem().Interface()
		}
	}

	return outsValue, nil
}
