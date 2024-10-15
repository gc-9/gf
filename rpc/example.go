package rpc

import (
	"net/http"
	"time"
)

type exampleService struct {
}

func (t *exampleService) Ping(ip string) string {
	return "pong"
}

type exampleServiceInterface interface {
	Ping(ip string) (string, error)
}

func newExampleClient(remoteUrl string) exampleServiceInterface {
	return &exampleClient{
		RemoteInject: NewRemoteInject(remoteUrl),
	}
}

type exampleClient struct {
	*RemoteInject
}

func (t *exampleClient) Ping(ip string) (string, error) {
	outs, err := t.Call(MethodName(0), []any{ip})
	if err != nil {
		return "", err
	}
	return RValue[string](outs[0]), nil
}

func ExampleRun() {
	// provide rpc service
	go func() {
		srv := &exampleService{}
		proxy := NewInvokeProxy(srv)

		http.HandleFunc("/rpc", proxy.HttpHandler)
		http.ListenAndServe(":8080", nil)
	}()

	time.Sleep(time.Second * 10)

	// call remote
	cc := newExampleClient("http://127.0.0.1:8080")
	cc.Ping("127.0.0.1")
}
