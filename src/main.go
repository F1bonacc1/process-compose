package main

import (
	"github.com/f1bonacc1/process-compose/src/cmd"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	cmd.Execute()
}
