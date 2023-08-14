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
curl -i -XPOST -H'Authorization: Bearer dolphin' http://localhost:4711/hello
```

## FAQ

### How to detect deploy failures from the http client?

Redeployster replies with HTTP Status 200 as soon as a deploy job starts because it then streams the deploy output. To access the exit code of a job, you can read the `Exit-Code` HTTP trailer.

Here is an example with _curl_ that saves the headers in a separate file and then uses _grep_ to exit with a non-zero code if the deployment fails:

```shell
curl -XPOST -H'Authorization: Bearer dolphin' http://localhost:4711/hello -D headers.txt
grep -q '^Exit-Code: 0\b' headers.txt
```

### How to expose redeployster to the Internet under HTTPS?

Redeployster currently doesn't have options to provide a certificate in order to listen directly on port 443.

It is meant to run on the same host as the services it needs to deploy. These Docker-managed services are most likely exposed to the HTTPS port via a reverse proxy like [Nginx](https://nginx.org), [Traefik](https://traefik.io/), [Caddy](https://caddyserver.com/) etc.

One typical scenario is to add a _forwarder_ service under the reverse-proxy. The _forwarder_ will then proxy requests to redeployster running directly on the host. See this example using Caddy as the forwarder, and assuming Traefik as the main reverse-proxy.

```yml
# docker-compose.yml
services:
  forwarder:
    image: caddy:2.6.4-alpine
    command: 'caddy reverse-proxy --from :3000 --to host.docker.internal:4711'
    extra_hosts:
      - "host.docker.internal:host-gateway"
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.forwarder.rule=Host(`deploy.example.com`)"
      - "traefik.http.services.forwarder.loadbalancer.server.port=3000"
      - "traefik.http.routers.forwarder.tls.certresolver=default"
```

Note: If you have a firewall, you might need to allow Docker's network interface to access the Redeployster's port (4711)

### Why not run redeployster within Docker?

We could also run it within a Docker container and let it handle the other containers by mounting the Docker socket. But redeployster actually needs to call _docker-compose_, and for this it needs access to the _docker-compose.yml_ file from the host.

## Development

Requirements: [go](https://golang.org)

Useful commands:

- Build the binary: `go build .`
- Build & run from source: `go run .`
- Format the code: `go fmt .`
