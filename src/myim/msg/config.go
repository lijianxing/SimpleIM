package main

import (
	"flag"
	"runtime"
	"time"

	"github.com/Terry-Mao/goconf"
)

var (
	gconf    *goconf.Config
	Conf     *Config
	confFile string
)

func init() {
	flag.StringVar(&confFile, "c", "./msg.conf", " set msg config file path")
}

type Config struct {
	// base section
	PidFile string `goconf:"base:pidfile"`
	Dir     string `goconf:"base:dir"`
	Log     string `goconf:"base:log"`
	MaxProc int    `goconf:"base:maxproc"`

	// mysql
	DbDsn     string `goconf:"mysql:dsn"`
	DbMaxOpen int    `goconf:"mysql:max_open"`
	DbMaxIdle int    `goconf:"mysql:max_idle"`

	// rpc
	RPCAddrs []string `goconf:"rpc:addrs:,"`

	// bucket
	Bucket int `goconf:"bucket:bucket"`

	// chat
	ChatMsgNum int           `goconf:"chat:msg_num"`
	ChatExpire time.Duration `goconf:"chat:expire:time"`
	Chat       int           `goconf:"chat:chat"`

	// monitor
	MonitorOpen  bool     `goconf:"monitor:open"`
	MonitorAddrs []string `goconf:"monitor:addrs:,"`
}

func NewConfig() *Config {
	return &Config{
		// base section
		PidFile: "/tmp/goim-router.pid",
		Dir:     "./",
		Log:     "./router-log.xml",
		MaxProc: runtime.NumCPU(),

		// mysql
		DbMaxOpen: 10,
		DbMaxIdle: 1,

		// rpc
		RPCAddrs: []string{"localhost:9090"},

		// bucket
		Bucket: runtime.NumCPU(),

		// chat
		Chat:       1024,
		ChatMsgNum: 20,
		ChatExpire: time.Minute * 20,
	}
}

// InitConfig init the global config.
func InitConfig() (err error) {
	Conf = NewConfig()
	gconf = goconf.New()
	if err = gconf.Parse(confFile); err != nil {
		return err
	}
	if err := gconf.Unmarshal(Conf); err != nil {
		return err
	}
	return nil
}

func ReloadConfig() (*Config, error) {
	conf := NewConfig()
	ngconf, err := gconf.Reload()
	if err != nil {
		return nil, err
	}
	if err := ngconf.Unmarshal(conf); err != nil {
		return nil, err
	}
	gconf = ngconf
	return conf, nil
}
