package config

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"go.uber.org/zap/zapcore"
)

type Server struct {
	Name    string          `yaml:"name"`    // 服务名
	Prefix  string          `yaml:"prefix"`  // 路由前缀
	Addr    string          `yaml:"addr"`    // 监听地址
	Url     string          `yaml:"url"`     // 外部地址
	Statics []*ServerStatic `yaml:"statics"` // 静态文件服务
	DocPath string          `yaml:"docPath"` // 文档地址

	Logger     Logger     `yaml:"logger"`     // 日志配置
	RequestLog RequestLog `yaml:"requestLog"` // 请求日志配置

	Acl  Acl        `yaml:"acl"`
	Auth AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
	CachePrefix string        `yaml:"cachePrefix"`
	Duration    time.Duration `yaml:"duration"`
	MaxDevices  int           `yaml:"maxDevices"`
}

type Logger struct {
	Path       string        `yaml:"path"`
	Level      zapcore.Level `yaml:"level"`
	MaxSize    int           `yaml:"maxSize"`
	MaxBackups int           `yaml:"maxBackups"`
	MaxAge     int           `yaml:"maxAge"`
}

// RequestLog configures HTTP access logging.
type RequestLog struct {
	DumpBody    bool     `yaml:"dumpBody"`
	IgnorePaths []string `yaml:"ignorePaths"`

	ignorePathRegexps []*regexp.Regexp
}

// Compile validates request-log path patterns once during server startup.
func (c *RequestLog) Compile() error {
	c.ignorePathRegexps = c.ignorePathRegexps[:0]
	for _, pattern := range c.IgnorePaths {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("compile requestLog.ignorePaths %q: %w", pattern, err)
		}
		c.ignorePathRegexps = append(c.ignorePathRegexps, re)
	}
	return nil
}

// ShouldLog reports whether a request should be recorded. Matched paths skip
// successful polling requests, while errors and 5xx responses always remain.
func (c *RequestLog) ShouldLog(path string, status int, requestErr error) bool {
	if requestErr != nil || status >= http.StatusInternalServerError {
		return true
	}
	for _, re := range c.ignorePathRegexps {
		if re.MatchString(path) {
			return false
		}
	}
	return true
}

type ServerStatic struct {
	Root string
	Path string
}

type Acl struct {
	SuperRoleKey string `yaml:"superRoleKey"` // 后台超管的role key
	AuthHeader   string `yaml:"authHeader"`

	AllowGuestPaths     []*regexp.Regexp `yaml:"allowGuestPaths"`
	IgnoreAuthPaths     []*regexp.Regexp `yaml:"ignoreAuthPaths"`
	IgnoreAclPaths      []*regexp.Regexp `yaml:"ignoreAclPaths"`
	IgnoreAuditLogPaths []*regexp.Regexp `yaml:"ignoreAuditLogPaths"`
}

func (c *Acl) AfterBind() {
	if c.SuperRoleKey == "" {
		c.SuperRoleKey = "super"
	}
	if c.AuthHeader == "" {
		c.AuthHeader = "X-Auth-Token"
	}
}
