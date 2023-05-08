package main

import (
    "fmt"
    "os/exec"
    "io"
    "net/http"
    "strings"
)


type Event struct {
    data []byte
}

type Bus chan chan *Event
type Service struct {
    bus Bus
    token string
}

type State map[string]Service

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

func deploy(name string) chan *Event {
    var ch = make(chan *Event)

    go func() {
        cmd := exec.Command("bash", "-c", fmt.Sprintf("echo deploying %s...  && sleep 2s && ls ~", name))
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

func manageService(name string, bus Bus) {
    go func() {
        var inProgress = false
        var deploymentEvents chan *Event

        var currentDeployListeners []chan *Event
        var nextDeployListeners []chan *Event

        for {
            select {
            case client := <-bus:
                if inProgress {
                    nextDeployListeners = append(nextDeployListeners, client)
                    client <- &Event{data: []byte("*** A deployment is currently in progress, queued\n")}
                } else {
                    currentDeployListeners = append(currentDeployListeners, client)
                    inProgress = true
                    deploymentEvents = deploy(name)
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
                        deploymentEvents = deploy(name)
                    } else {
                        currentDeployListeners = nil
                        inProgress = false
                        deploymentEvents = nil
                    }
                }
            }
        }
    }()
}

func extractBearerToken(r *http.Request) string {
    authHeader := r.Header.Get("Authorization")
    authHeaderParts := strings.Split(authHeader, " ")
    if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
        return ""
    }

    return authHeaderParts[1]
}

func makeHandler(state State) func(http.ResponseWriter, *http.Request) {
    return func (w http.ResponseWriter, r *http.Request) {
        fmt.Println("Handling ", r.URL.Path)
        service, ok := state[strings.TrimLeft(r.URL.Path, "/")]

        if !ok {
            http.NotFound(w, r)
            return
        }

        if extractBearerToken(r) != service.token {
            // Return 404 instead of 403 to reduce exposure
            http.NotFound(w, r)
            return
        }

        ch := make(chan *Event)
        service.bus <- ch

        for e := range ch {
            w.Write(e.data)
            if f, ok := w.(http.Flusher); ok {
                f.Flush()
            }
        }
    }
}

func main() {
    state := State{}
    state["service1"] = Service{
        bus: make(chan chan *Event),
        token: "dolphin",
    }
    state["service2"] = Service{
        bus: make(chan chan *Event),
        token: "beaver",
    }

    for name, service := range state {
        manageService(name, service.bus)
    }

    http.HandleFunc("/", makeHandler(state))

    fmt.Println("Listening on http://0.0.0.0:4711")
    http.ListenAndServe(":4711", nil)
}
