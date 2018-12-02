// Copyright © 2014 Terry Mao, LiuDing All rights reserved.
// This file is part of gopush-cluster.

// gopush-cluster is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// gopush-cluster is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with gopush-cluster.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"runtime"
	"strconv"
	"time"

	"github.com/Terry-Mao/goconf"
)

var (
	gconf    *goconf.Config
	Conf     *Config
	confFile string
)

func init() {
	flag.StringVar(&confFile, "c", "./logic.conf", " set logic config file path")
}

type Config struct {
	// base section
	PidFile string `goconf:"base:pidfile"`
	Dir     string `goconf:"base:dir"`
	Log     string `goconf:"base:log"`
	MaxProc int    `goconf:"base:maxproc"`

	RPCAddrs         []string      `goconf:"base:rpc.addrs:,"`
	HTTPAddrs        []string      `goconf:"base:http.addrs:,"`
	HTTPReadTimeout  time.Duration `goconf:"base:http.read.timeout:time"`
	HTTPWriteTimeout time.Duration `goconf:"base:http.write.timeout:time"`

	// router
	SessionExpireSec int `goconf:"router:session_expire_ts"`

	// redis
	RedisAddr            string        `goconf:"redis:addr"`
	RedisPoolMaxActive   int           `goconf:"redis:pool.max_active"`
	RedisPoolMaxIdle     int           `goconf:"redis:pool.max_idle"`
	RedisPoolIdleTimeout time.Duration `goconf:"redis:pool.idle_timeout:time"`

	// comet RPC
	CometRPCAddrs map[int32]string `-`
	RoutineSize   uint64           `goconf:"comet:routine.size"`
	RoutineChan   int              `goconf:"comet:routine.chan"`

	// monitor
	MonitorOpen  bool     `goconf:"monitor:open"`
	MonitorAddrs []string `goconf:"monitor:addrs:,"`
}

func NewConfig() *Config {
	return &Config{
		// base section
		PidFile:       "/tmp/goim-logic.pid",
		Dir:           "./",
		Log:           "./logic-log.xml",
		MaxProc:       runtime.NumCPU(),
		HTTPAddrs:     []string{"7172"},
		CometRPCAddrs: make(map[int32]string),

		// comet
		RoutineSize: 16,
		RoutineChan: 64,

		// redis
		RedisAddr:            "localhost:6379",
		RedisPoolMaxActive:   0,
		RedisPoolMaxIdle:     0,
		RedisPoolIdleTimeout: 60 * time.Second,
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

	// comet
	var serverIDi int64
	for _, serverID := range gconf.Get("comets").Keys() {
		addr, err := gconf.Get("comets").String(serverID)
		if err != nil {
			return err
		}
		serverIDi, err = strconv.ParseInt(serverID, 10, 32)
		if err != nil {
			return err
		}
		Conf.CometRPCAddrs[int32(serverIDi)] = addr
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
