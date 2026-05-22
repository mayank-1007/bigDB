package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"bigtable/internal/api"
	"bigtable/internal/db"
)

func main() {
	dataDir := flag.String("data-dir", "./data", "path to database data directory")
	putKey := flag.String("put-key", "", "key to write")
	putValue := flag.String("put-value", "", "value to write")
	getKey := flag.String("get-key", "", "key to read")
	delKey := flag.String("del-key", "", "key to delete")
	compact := flag.Bool("compact", false, "run compaction")
	pprofAddr := flag.String("pprof", "", "pprof listen address, e.g. :6060")
	serve := flag.Bool("serve", false, "start the HTTP API server for the visualizer")
	httpAddr := flag.String("http", ":8080", "HTTP listen address for the visualizer API")

	flag.Parse()

	opts := db.DefaultOptions()
	opts.DataDir = *dataDir

	engine, err := db.Open(opts)
	if *serve {
		srv := api.NewServer(engine)

		go func() {
			log.Printf("API listening on http://localhost%s", *httpAddr)
			if err := http.ListenAndServe(*httpAddr, srv.Handler()); err != nil {
				log.Printf("api server stopped: %v", err)
			}
		}()

		if *pprofAddr != "" {
			go func() {
				log.Printf("pprof listening on http://localhost%s/debug/pprof/", *pprofAddr)
				if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
					log.Printf("pprof server stopped: %v", err)
				}
			}()
		}

		fmt.Println("database opened successfully")
		select {}
	}
	
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	defer func() {
		if cerr := engine.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "close db: %v\n", cerr)
		}
	}()

	if *pprofAddr != "" {
		go func() {
			fmt.Printf("pprof running → http://localhost%s/debug/pprof/\n", *pprofAddr)

			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Printf("pprof server stopped: %v", err)
			}
		}()
	}

	if *putKey != "" {
		if err := engine.Put([]byte(*putKey), []byte(*putValue)); err != nil {
			log.Fatalf("put: %v", err)
		}
		fmt.Println("put ok")
	}

	if *delKey != "" {
		if err := engine.Delete([]byte(*delKey)); err != nil {
			log.Fatalf("delete: %v", err)
		}
		fmt.Println("delete ok")
	}

	if *getKey != "" {
		value, ok, err := engine.Get([]byte(*getKey))
		if err != nil {
			log.Fatalf("get: %v", err)
		}

		if !ok {
			fmt.Println("not found")
		} else {
			fmt.Printf("%s => %s\n", *getKey, string(value))
		}
	}

	if *compact {
		if err := engine.Compact(); err != nil {
			log.Fatalf("compact: %v", err)
		}
		fmt.Println("compaction ok")
	}

	// KEEP PROCESS ALIVE ONLY FOR PPROF
	if *pprofAddr != "" {
		fmt.Println("press CTRL+C to stop")

		select {}
	}
}