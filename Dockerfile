FROM golang:1.26.3

RUN mkdir /app
WORKDIR /app

RUN apt-get update && apt-get upgrade -y && apt-get install -y make git

# Cache de deps Go
COPY go.mod go.sum ./
RUN go mod download

# Instal·lar air
RUN go install github.com/air-verse/air@latest

CMD ["air", "-c", ".air.toml"]
