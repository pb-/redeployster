# redeployster


## Rationale

Sometimes Kubernetes is the answer, but other times not. _redeployster_ recreates containers in scrappy, non-enterprise docker-compose setups. It's only purpose is to run `docker-compose up -d $SERVICE` and to expose that functionality through HTTP so that your build pipelines can trigger it remotely.

## Development

Requirements: [go](https://golang.org)

Useful commands:

- Build the binary: `go build -o redeployster main.go`
- Build & run from source: `go run main.go`

## TODO

 * Respond with 500 when deployment exit code is not zero
   * Not possible, headers already sent!
   * HTTP trailer?
 * Add authentication (static token)
 * Read docker-compose.yml and set up one manageService() per service
 * Get service name from request URI
