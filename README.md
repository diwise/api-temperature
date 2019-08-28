# Introduction 

This service is responsible for storing temperature telemetry and provide it to consumers via an API.

# Getting Started

Until we have a shared container registry it is required that you pull the ingress-trafikverket repository and use Docker to build and tag an image for the ingress-trafikverket service. It is also required to register with Trafikverket for an API key.

Then, execute the `docker-compose` command shown below.

# Building and tagging with Docker

`docker build -f deployments/Dockerfile -t iot-for-tillgenglighet/api-temperature:latest .`

# Build for local testing with Docker Compose

`docker-compose -f ./deployments/docker-compose.yml build`

# Running locally with Docker Compose

`TFV_API_AUTH_KEY=<insert your API key here> docker-compose -f ./deployments/docker-compose.yml up`

The ingress service will exit fatally and restart a couple of times until the RabbitMQ container is properly initialized and ready to accept connections. This is to be expected.

To clean up the environment properly after testing it is advisable to run `docker-compose down -v`
