package logs

// 配置文件模块

import (
	"io/ioutil"
	"log"

	"go.uber.org/zap"

	"github.com/naoina/toml"
	"github.com/naoina/toml/ast"
)

// CommonConfig Common
type CommonConfig struct {
	Version  string
	IsDebug  bool
	LogLevel string
	LogPath  string
}

// EchoConf echo config struct
type EchoConf struct {
	Addr string
}

// Config ...
type Config struct {
	Common *CommonConfig
	EchoC  *EchoConf
}

// Conf ...
var Conf = &Config{}

// LoadConfig ...
func LoadConfig() {
	// init the new config params
	initConf()

	contents, err := ioutil.ReadFile("goetcd.toml")
	if err != nil {
		log.Fatal("[FATAL] load goetcd.toml: ", err)
	}
	tbl, err := toml.Parse(contents)
	if err != nil {
		log.Fatal("[FATAL] parse goetcd.toml: ", err)
	}
	// parse common config
	parseCommon(tbl)
	// init log
	InitLogger()

	// parse Echo config
	parseEcho(tbl)

	Logger.Info("LoadConfig", zap.Any("Config", Conf))
}

func initConf() {
	Conf = &Config{
		Common: &CommonConfig{},
		EchoC:  &EchoConf{},
	}
}

func parseCommon(tbl *ast.Table) {
	if val, ok := tbl.Fields["common"]; ok {
		subTbl, ok := val.(*ast.Table)
		if !ok {
			log.Fatalln("[FATAL] : ", subTbl)
		}

		err := toml.UnmarshalTable(subTbl, Conf.Common)
		if err != nil {
			log.Fatalln("[FATAL] parseCommon: ", err, subTbl)
		}
	}
}

func parseEcho(tbl *ast.Table) {
	if val, ok := tbl.Fields["ech"]; ok {
		subTbl, ok := val.(*ast.Table)
		if !ok {
			log.Fatalln("[FATAL] : ", subTbl)
		}

		err := toml.UnmarshalTable(subTbl, Conf.EchoC)
		if err != nil {
			log.Fatalln("[FATAL] parseEcho: ", err, subTbl)
		}
	}
}
