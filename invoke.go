package main

import (
	"fmt"
	"io"
	"os/exec"
)

func runCmd(prg string, cmdargs []string, ch chan *Event) int {
	cmd := exec.Command(prg, cmdargs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("problem")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println("problem")
	}

	if err := cmd.Start(); err != nil {
		fmt.Println("problem")
	}

	<-forwardOutput(stdout, ch)
	<-forwardOutput(stderr, ch)

	cmd.Wait()
	return cmd.ProcessState.ExitCode()
}

func forwardOutput(r io.Reader, ch chan *Event) chan bool {
	var done = make(chan bool)
	go func() {
		buffer := make([]byte, 100)
		for {
			n, err := r.Read(buffer)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])
				ch <- &Event{data: data}
			}

			if err != nil {
				done <- true
				return
			}
		}
	}()

	return done
}
