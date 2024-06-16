# syntax=docker/dockerfile:1
FROM golang:1.20

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o /faceit-task

EXPOSE 8080 9090

CMD [ "/faceit-task" ]
