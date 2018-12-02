package main

import (
	"encoding/json"
	"net/http"

	log "github.com/thinkboy/log4go"
)

type Monitor struct {
}

// StartPprof start http monitor.
func InitMonitor(binds []string) {
	m := new(Monitor)
	monitorServeMux := http.NewServeMux()
	monitorServeMux.HandleFunc("/monitor/ping", m.Ping)
	monitorServeMux.HandleFunc("/monitor/stat", m.Stat)
	for _, addr := range binds {
		go func(bind string) {
			log.Info("start monitor listen: \"%s\"", addr)
			if err := http.ListenAndServe(bind, monitorServeMux); err != nil {
				log.Error("http.ListenAndServe(\"%s\", pprofServeMux) error(%v)", addr, err)
				panic(err)
			}
		}(addr)
	}
}

// monitor ping
func (m *Monitor) Ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

// monitor stat
func (m *Monitor) Stat(w http.ResponseWriter, r *http.Request) {
	var (
		err error
		b   []byte
		res = map[string]interface{}{"ret": OK}
	)
	switch r.Method {
	case "GET":
		res["data"] = DefaultStat.Info()
	case "DELETE":
		DefaultStat.Reset()
	}
	if b, err = json.Marshal(res); err != nil {
		log.Error("json.Marshal(%v) error(%v)", res, err)
		return
	}
	w.Write(b)
}
