package main

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type Event struct {
	data []byte
}

type Bus chan chan *Event
type Service struct {
	bus         Bus
	composeFile string
	token       string
}

type State map[string]Service

func deploy(name string, composeFile string) chan *Event {
	var ch = make(chan *Event)

	go func() {
		defer close(ch)

		exitCode := runCmd(
			"docker-compose",
			[]string{"-f", composeFile, "up", "--pull", "always", "-d", name},
			ch,
		)

		ch <- &Event{data: []byte(fmt.Sprintf("*** Deployment command finished with exit code %d\n", exitCode))}
	}()
	return ch
}

func manageService(name string, composeFile string, bus Bus) {
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
					deploymentEvents = deploy(name, composeFile)
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
						deploymentEvents = deploy(name, composeFile)
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

func isValidToken(suppliedToken string, correctToken string) bool {
	return subtle.ConstantTimeCompare([]byte(suppliedToken), []byte(correctToken)) == 1
}

func makeHandler(s *State) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Handling", r.URL.Path)

		name := strings.TrimLeft(r.URL.Path, "/")

		service, ok := (*s)[name]
		if !ok {
			// Reload the state in case the service was recently added
			loadState(s)
			service, ok = (*s)[name]
			if !ok {
				http.NotFound(w, r)
				return
			}
		}

		if !isValidToken(extractBearerToken(r), service.token) {
			// Return 404 instead of 403 to reduce exposure
			http.NotFound(w, r)
			return
		}

		// Reload the state in case the service was recently removed
		loadState(s)
		if _, ok = (*s)[name]; !ok {
			w.WriteHeader(http.StatusGone)
			fmt.Fprintf(w, http.StatusText(http.StatusGone))
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

func loadState(s *State) error {
	output, err := listContainers()
	if err != nil {
		return err
	}

	serviceSet := map[string]bool{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}

		name := fields[0]

		if service, ok := (*s)[name]; ok {
			serviceSet[name] = true
			service.composeFile = fields[1]
			service.token = fields[2]
		} else {
			serviceSet[name] = true
			service := Service{
				bus:         make(chan chan *Event),
				composeFile: fields[1],
				token:       fields[2],
			}

			(*s)[name] = service
			manageService(name, service.composeFile, service.bus)
			fmt.Printf("Service %s configured\n", name)
		}
	}

	for name, service := range *s {
		if _, ok := serviceSet[name]; !ok {
			close(service.bus)
			delete(*s, name)
			fmt.Printf("Service %s unmounted\n", name)
		}
	}

	return nil
}

func main() {
	state := State{}
	if err := loadState(&state); err != nil {
		fmt.Fprintf(os.Stderr, "%e", err)
		os.Exit(1)
	}

	http.HandleFunc("/", makeHandler(&state))

	port := 4711
	log.Printf("Trying to listen on http://0.0.0.0:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
