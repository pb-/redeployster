package main

import (
    "fmt"
    "os/exec"
    "io"
    "net/http"
)


type Event struct {
    data []byte
}

var control chan chan *Event

func forwardOutput(r io.Reader, ch chan *Event) chan bool {
    var done = make(chan bool)
    go func() {
        buffer := make([]byte, 100);
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

func deploy() chan *Event {
    var ch = make(chan *Event)

    go func() {
        cmd := exec.Command("bash", "-c", "echo working && sleep 2s && ls ~")
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

        <- forwardOutput(stdout, ch)
        <- forwardOutput(stderr, ch)

        cmd.Wait()
        ch <- &Event{data: []byte(fmt.Sprintf("*** Deployment command finished with exit code %d\n", cmd.ProcessState.ExitCode()))}
        close(ch)
    }()

    return ch
}

func manageService() chan chan *Event {
    var control = make(chan chan *Event)

    go func() {
        var inProgress = false
        var deploymentEvents chan *Event

        var currentDeployListeners []chan *Event
        var nextDeployListeners []chan *Event

        for {
            select {
            case client := <-control:
                if inProgress {
                    nextDeployListeners = append(nextDeployListeners, client)
                    client <- &Event{data: []byte("*** A deployment is currently in progress, queued\n")}
                } else {
                    currentDeployListeners = append(currentDeployListeners, client)
                    inProgress = true
                    deploymentEvents = deploy()
                }
            case e, ok := <-deploymentEvents:
                if ok {
                    for _, ch := range currentDeployListeners {
                        ch <- e
                    }
                    for _, ch := range nextDeployListeners {
                        ch <- e
                    }
                } else {
                    for _, ch := range currentDeployListeners {
                        close(ch)
                    }

                    if nextDeployListeners != nil {
                        currentDeployListeners = nextDeployListeners
                        nextDeployListeners = nil
                        deploymentEvents = deploy()
                    } else {
                        currentDeployListeners = nil
                        inProgress = false
                        deploymentEvents = nil
                    }
                }
            }
        }
    }()

    return control
}

func handle(w http.ResponseWriter, r *http.Request) {
    ch := make(chan *Event)
    control <- ch

    for e := range ch {
        // w.Write(e)
        //fmt.Fprintln(w, e)
        w.Write(e.data)
        if f, ok := w.(http.Flusher); ok {
            f.Flush()
        }
    }
}

func main() {
    control = manageService()
    http.HandleFunc("/", handle)
    http.ListenAndServe(":4711", nil)
}
