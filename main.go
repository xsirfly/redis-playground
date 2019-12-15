package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/gomodule/redigo/redis"
)

var addr = flag.String("addr", ":80", "http service address")

var modules = [6]string{"redis", "redislabs/rejson", "redislabs/redisearch", "redislabs/rebloom", "redislabs/redisgraph", "redislabs/redisml"}

// var modules = [1]string{"redis"}

var dockerAPI = newDockerAPI()

func setupModules() {
	/* Copy static content into module folder. */
	wd, _ := os.Getwd()
	staticFiles, err := ioutil.ReadDir(path.Join(wd, "static"))
	if err != nil {
		log.Fatal(err)
	}

	for _, m := range modules {
		moduleDir := path.Join(wd, "static", m)
		os.MkdirAll(moduleDir, os.ModePerm)
		for _, f := range staticFiles {
			in, _ := os.Open(path.Join(wd, "static", f.Name()))
			out, _ := os.Create(path.Join(moduleDir, f.Name()))
			io.Copy(out, in)
			in.Close()
			out.Close()
		}
	}

	createAutoCompleteLists()
	registerWebHandlers()
}

func registerWebHandlers() {
	// Static files
	wd, _ := os.Getwd()
	staticDir := path.Join(wd, "/static")
	fs := http.FileServer(http.Dir(staticDir))
	http.Handle("/", fs)

	for _, m := range modules {
		// Websocket
		http.HandleFunc("/"+m+"/ws", func(w http.ResponseWriter, r *http.Request) {
			serveWs(w, r)
		})
	}
}

func createAutoCompleteLists() {
	for _, m := range modules {
		if containerID, err := dockerAPI.RunContainer(m); err == nil {
			if containerIP, err := dockerAPI.GetContainerIP(containerID); err == nil {
				// Get list of commands supported by Redis.
				if c, err := redis.DialTimeout("tcp", containerIP+":6379", redisConnectTimeout, redisReadTimeout, redisWriteTimeout); err == nil {
					if commands, err := RedisCommandList(c); err == nil {
						j, _ := json.Marshal(commands)
						wd, _ := os.Getwd()
						staticDir := path.Join(wd, "/static", m)
						os.MkdirAll(staticDir, os.ModePerm)
						autocompleteFile := path.Join(staticDir, "/autocomplete.js")
						f, _ := os.Create(autocompleteFile)
						f.WriteString("var redisCommands = ")
						f.Write(j)
						f.WriteString(";")
						f.Close()
					}
					c.Close()
				}
			}
			dockerAPI.RemoveContainer(containerID)
		}
	}
}

func main() {
	flag.Parse()
	setupModules()
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
