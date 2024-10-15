package rpc

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/gc-9/gf/errors"
	"io"
	"net/http"
	"reflect"
)

func gobEncode(a any) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(a)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gobDecode(buf []byte, a any) error {
	dec := gob.NewDecoder(bytes.NewBuffer(buf))
	return dec.Decode(a)
}

func callHandler(c *reflectHandler, argsBuf [][]byte) ([][]byte, error) {
	var err error
	args := make([]reflect.Value, len(c.inTypes))
	for i, argBuf := range argsBuf {
		inType := c.inTypes[i]

		isPtr := inType.Kind() == reflect.Ptr
		if isPtr {
			inType = inType.Elem()
		}
		arg := reflect.New(inType).Interface()
		err := gobDecode(argBuf, arg)
		if err != nil {
			return nil, err
		}
		args[i] = reflect.ValueOf(arg)
		if !isPtr {
			args[i] = args[i].Elem()
		}
	}
	outs := c.caller.Call(args)
	outsBuf := make([][]byte, len(outs))
	for k, out := range outs {

		// check out is nil
		if !out.IsValid() || out.Interface() == nil {
			outsBuf[k] = nil
			continue
		}

		if out.Type().Implements(errorType) {
			value := fmt.Sprintf("%v", out.Interface())
			outsBuf[k], err = gobEncode(value)
			if err != nil {
				return nil, err
			}
			continue
		}
		outsBuf[k], err = gobEncode(out.Interface())
		if err != nil {
			return nil, err
		}
	}
	return outsBuf, nil
}

type InvokeProxy struct {
	methodMap map[string]*reflectHandler
}

func NewInvokeProxy(cm any) *InvokeProxy {
	tValue := reflect.ValueOf(cm)
	tType := tValue.Type()

	methodMap := make(map[string]*reflectHandler)
	for i := 0; i < tType.NumMethod(); i++ {
		method := tType.Method(i)

		funcNoReceiver := tValue.MethodByName(method.Name)
		methodMap[method.Name] = parseMethod(funcNoReceiver)
	}
	return &InvokeProxy{
		methodMap: methodMap,
	}
}

func (c *InvokeProxy) Call(method string, args [][]byte) ([][]byte, error) {
	v, ok := c.methodMap[method]
	if !ok {
		panic(errors.New("method not found"))
	}
	return callHandler(v, args)
}

func (c *InvokeProxy) HttpHandler(writer http.ResponseWriter, request *http.Request) {
	method := request.URL.Query().Get("method")
	buf, err := io.ReadAll(request.Body)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	if method == "" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	var args [][]byte
	err = gobDecode(buf, &args)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	outsBuf, err := c.Call(method, args)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	buf, err = gobEncode(outsBuf)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.WriteHeader(http.StatusOK)
	writer.Write(buf)
}
