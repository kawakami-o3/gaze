package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
)

var watchTarget string
var command string

func addRetry(w *fsnotify.Watcher) {
	w.Remove(watchTarget)
	t := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-t.C:
			err := w.Add(watchTarget)
			if err == nil {
				t.Stop()
				return
			} else {
				fmt.Println("retry...")
			}
		}
	}
}

func main() {
	flag.Parse()
	args := flag.Args()

	//watchTarget = strings.Join(args, " ")
	watchTarget = args[0]
	command = watchTarget
	fmt.Println(command)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					fmt.Println("event panic")
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}

				addRetry(watcher)
			case err, ok := <-watcher.Errors:
				if !ok {
					fmt.Println("error panic")
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	addRetry(watcher)
	<-done
}
