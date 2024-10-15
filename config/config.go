package config

import (
	"flag"
	"fmt"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/i18n"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path"
	"regexp"
	"time"
)

// log setting
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
}

var (
	conf     = "./config.yaml"
	showHelp = false
)

func Parse() *Config {
	flag.StringVar(&conf, "conf", "./config.yaml", "config file")
	flag.BoolVar(&showHelp, "help", false, "show help")
	flag.Parse()

	if showHelp || conf == "" {
		flag.Usage()
		os.Exit(0)
	}

	c, err := NewConfig(conf)
	if err != nil {
		fmt.Printf("NewConfig error: %v\n", err)
		os.Exit(1)
	}

	return c
}

func NewI18n() (i18n.I18n, error) {
	var i18nMap i18n.I18nMap
	i18nPath := path.Join(path.Dir(conf), "i18n.yaml")
	err := unmarshalFile(i18nPath, &i18nMap)
	if err != nil {
		return nil, err
	}
	return &i18nMap, nil
}

func unmarshalFile(filepath string, conf interface{}) error {
	buf, err := os.ReadFile(filepath)
	if err != nil {
		return errors.Wrap(err, "read file error")
	}

	err = yaml.Unmarshal(buf, conf)
	return errors.Wrap(err, "unmarshal file error")
}

func NewConfig(filename string) (*Config, error) {
	c := new(Config)
	err := c.loadFile(filename)
	if err != nil {
		return nil, err
	}
	return c, nil
}

type Config struct {
	mapNode map[string]*yaml.Node
	App     *App
}

type afterBinder interface {
	AfterBind()
}

func (c *Config) loadFile(filename string) (err error) {
	var root yaml.Node
	err = unmarshalFile(filename, &root)
	if err != nil {
		return
	}

	if root.Kind != yaml.DocumentNode {
		return errors.Errorf("config rootKind is %v not DocumentNode", yaml.DocumentNode)
	}

	if len(root.Content) != 1 {
		return errors.New("config len(root.Content) != 1")
	}

	if root.Content[0].Kind != yaml.MappingNode {
		return errors.Errorf("config root.Content[0] is %v not MappingNode", yaml.MappingNode)
	}

	c.mapNode = make(map[string]*yaml.Node)
	c.mapNode[""] = &root

	contents := root.Content[0].Content
	for i := 0; i < len(contents); i += 2 {
		key := contents[i].Value
		valueNode := contents[i+1]
		c.mapNode[key] = valueNode
	}

	// auto bind app
	var app App
	err = c.Get("app", &app)
	return err
}

func (c *Config) Get(part string, value any) error {
	v, ok := c.mapNode[part]
	if !ok {
		return errors.Errorf("config part %s not exist", part)
	}

	err := v.Decode(value)
	if err != nil {
		return errors.Wrapf(err, "decode value %s failed", part)
	}

	if b, ok := value.(afterBinder); ok {
		b.AfterBind()
	}
	return nil
}

func (c *Config) MustGet(part string, value any) {
	err := c.Get(part, value)
	if err != nil {
		panic(err)
	}
}

func Get[T any](c *Config, part string) (*T, error) {
	var t T
	err := c.Get(part, &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type App struct {
	Env  string `yaml:"env"`
	Name string `yaml:"name"`
}

func (t *App) IsDev() bool {
	return t.Env == "dev"
}

func (t *App) IsOnline() bool {
	return t.Env == "online"
}

type Database struct {
	DSN         string `yaml:"dsn"`
	TablePrefix string `yaml:"tablePrefix"` // 表前缀
	ShowSql     bool   `yaml:"showSql"`

	ConnMaxLifetime *time.Duration `yaml:"connMaxLifetime"`
	//ConnMaxIdleTime *time.Duration `yaml:"connMaxIdleTime"`
	MaxOpenConns int `yaml:"maxOpenConns"`
	MaxIdleConns int `yaml:"maxIdleConns"`
}

type Redis struct {
	//redis.Options
	Addr     string `yaml:"addr"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type Crypto struct {
	Key string `yaml:"key"`
}

type Generate struct {
	Output         string           `yaml:"output"`
	TablePrefix    string           `yaml:"tablePrefix"`
	IncludedTables []*regexp.Regexp `yaml:"includedTables"`
	IgnoreTables   []*regexp.Regexp `yaml:"ignoreTables"`
}
