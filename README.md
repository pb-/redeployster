# redeployster


## Rationale

Sometimes Kubernetes is the answer, but other times not. _redeployster_ recreates containers in scrappy, non-enterprise docker-compose setups. It's only purpose is to run `docker-compose up -d $SERVICE` and to expose that functionality through HTTP so that your build pipelines can trigger it remotely.
