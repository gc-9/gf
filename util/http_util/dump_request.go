package http_util

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

func DumpRequestForm(req *http.Request) (string, error) {
	if !strings.HasPrefix(req.Header.Get("content-type"), "multipart/form-data") {
		err := req.ParseForm()
		if err != nil {
			return "", err
		}

		if len(req.Form) > 0 {
			req.Form.Encode()
			return urlValues(req.Form), nil
		}
	}

	err := req.ParseMultipartForm(32 << 20)
	if err != nil {
		return "", err
	}

	body := ""
	if len(req.MultipartForm.Value) > 0 {
		body = urlValues(req.MultipartForm.Value)
	}

	if len(req.MultipartForm.File) > 0 {
		for k, files := range req.MultipartForm.File {
			var names []string
			for _, f := range files {
				names = append(names, f.Filename)
			}
			if len(body) > 0 {
				body += fmt.Sprintf("&%s={_file_}:%s", k, strings.Join(names, ","))
			} else {
				body += fmt.Sprintf("%s={_file_}:%s", k, strings.Join(names, ","))
			}
		}
	}
	return body, nil
}

func urlValues(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf strings.Builder
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		keyEscaped := k
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(v)
		}
	}
	return buf.String()
}

type BodyDumpResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *BodyDumpResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *BodyDumpResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *BodyDumpResponseWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *BodyDumpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}
