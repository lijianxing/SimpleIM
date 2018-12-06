package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/thinkboy/log4go"
)

const (
	OK = 1
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
			log.Info("start monitor listen: \"%s\"", bind)
			if err := http.ListenAndServe(bind, monitorServeMux); err != nil {
				log.Error("http.ListenAndServe(\"%s\", pprofServeMux) error(%v)", bind, err)
				panic(err)
			}
		}(addr)
	}
}

// monitor ping
func (m *Monitor) Ping(w http.ResponseWriter, r *http.Request) {
	if err := logicRpcClient.Available(); err != nil {
		http.Error(w, fmt.Sprintf("ping rpc error(%v)", err), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("ok"))
}

// monitor stat
func (m *Monitor) Stat(w http.ResponseWriter, r *http.Request) {
	var (
		err  error
		b    []byte
		res  = map[string]interface{}{"ret": OK}
		stat = DefaultServer.Stat
	)
	switch r.Method {
	case "GET":
		res["data"] = stat.Info()
	case "DELETE":
		stat.Reset()
	}
	if b, err = json.Marshal(res); err != nil {
		log.Error("json.Marshal(%v) error(%v)", res, err)
		return
	}
	w.Write(b)
}
