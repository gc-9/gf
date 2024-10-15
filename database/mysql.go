package database

import (
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/database/xorm_log"
	"github.com/gc-9/gf/logger"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
)

func NewDB(conf *config.Database) *xorm.Engine {

	engine, err := xorm.NewEngine("mysql", conf.DSN)
	if err != nil {
		log.Panic(err)
	}

	if conf.ConnMaxLifetime != nil {
		engine.SetConnMaxLifetime(*conf.ConnMaxLifetime)
	}
	if conf.MaxOpenConns > 0 {
		engine.SetMaxOpenConns(conf.MaxOpenConns)
	}
	if conf.MaxIdleConns > 0 {
		engine.SetMaxIdleConns(conf.MaxIdleConns)
	}

	engine.SetTableMapper(names.NewPrefixMapper(names.SnakeMapper{}, conf.TablePrefix))
	//engine.SetColumnMapper(names.SnakeMapper{})

	// timezone
	engine.TZLocation, _ = time.LoadLocation("Asia/Shanghai")

	// log adapt
	l := xorm_log.NewZapLoggerAdapt(logger.Logger())
	l.ShowSQL(conf.ShowSql)
	engine.SetLogger(l)

	return engine
}
