# redeployster


## Rationale

Sometimes Kubernetes is the answer, but other times not. _redeployster_ recreates containers in scrappy, non-enterprise docker-compose setups. It's only purpose is to run `docker-compose up -d $SERVICE` and to expose that functionality through HTTP so that your build pipelines can trigger it remotely.

## Usage

Attach a `redeployster.token` label to the services you want to make deployable:

```yml
# docker-compose.yml
version: "3"
services:
  hello:
    image: hello-world
    labels:
      - 'redeployster.token=dolphin'
```

Make sure that the service got launched manually once so the container(s) have the label:

```shell
docker-compose up -d
```

The deployable service is now configured:

- The name of the docker-compose service will be a new exposed path on redeployster: call `POST /hello` here to deploy the _hello_ service.
- The token will be required by reployster to trigger the deploy: add `Authorization: Bearer dolphin` http header to the call.
- Adding a token also acts as an opt-in flag for a service to be deployable. Without a token, redeployster will ignore the service.

### Try

```bash
go run .

# In another shell:
curl -i \
  -XPOST \
  -H'Authorization: Bearer dolphin' \
  http://localhost:4711/hello
```

## Development

Requirements: [go](https://golang.org)

Useful commands:

- Build the binary: `go build .`
- Build & run from source: `go run .`
- Format the code: `go fmt .`

## TODO

 * Respond with 500 when deployment exit code is not zero
   * Not possible, headers already sent!
   * HTTP trailer?
