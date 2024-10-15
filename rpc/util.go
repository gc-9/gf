package rpc

import (
	"runtime"
	"strings"
)

// n=0 current, >= 1 is fathers...
func MethodName(n int) string {
	pc, _, _, _ := runtime.Caller(n + 1)
	functionName := runtime.FuncForPC(pc).Name()
	return functionName[strings.LastIndex(functionName, ".")+1:]
}

func RValue[T any](a any) T {
	if a == nil {
		var zero T
		return zero
	}
	return a.(T)
}
