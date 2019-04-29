package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
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

// chWriter receives outputs from exec.Cmd.
type chWriter struct {
	ch chan string
}

func newChWriter(ch chan string) *chWriter {
	return &chWriter{
		ch: ch,
	}
}

func (c *chWriter) Write(p []byte) (n int, err error) {
	c.ch <- string(p)
	return len(p), nil
}

type Command struct {
	cmdStr string
	cmd    *exec.Cmd
	out    chan string
}

func NewCommand(cmd string) *Command {
	return &Command{
		cmdStr: cmd,
		cmd:    nil,
		out:    make(chan string, 10),
	}
}

func (c *Command) Run() {
	c.cmd = exec.Command(c.cmdStr)
	writer := newChWriter(c.out)
	c.cmd.Stdout = writer

	// This goroutine will stop when cmd is killed.
	go func() {
		c.cmd.Run()
	}()
}

func (c *Command) Kill() {
	c.cmd.Process.Kill()
	c.cmd.Wait()
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

	cmdTarget := NewCommand(command)
	cmdTarget.Run()

	done := make(chan bool)
	go func() {
		for {
			select {
			case s := <-cmdTarget.out:
				fmt.Print(s)
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}

				cmdTarget.Kill()
				addRetry(watcher)
				cmdTarget.Run()
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				cmdTarget.Kill()
				log.Println("error:", err)
			}
		}
	}()

	addRetry(watcher)
	<-done
}
