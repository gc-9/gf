package config

import "regexp"

type Server struct {
	Prefix  string          `yaml:"prefix"`  // 路由前缀
	Addr    string          `yaml:"addr"`    // 监听地址
	Url     string          `yaml:"url"`     // 外部地址
	Statics []*ServerStatic `yaml:"statics"` // 静态文件服务
	DocPath string          `yaml:"docPath"` // 文档地址

	Logger   Logger `yaml:"logger"`   // 日志配置
	DumpBody bool   `yaml:"dumpBody"` // 请求日志 是否打印body

	Acl Acl `yaml:"acl"`
}

type Logger struct {
	Path string `yaml:"path"`
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
		c.SuperRoleKey = "admin"
	}
	if c.AuthHeader == "" {
		c.AuthHeader = "X-Auth-Token"
	}
}
