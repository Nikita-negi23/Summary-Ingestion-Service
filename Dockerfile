# Basic syntax of writing a Dockerfile for a Go application
FROM golang:1.24.1

WORKDIR /app
COPY . .
RUN go mod download
# RUN go build -o server

EXPOSE 8080
# CMD ["./server"]

CMD ["go", "run", "main.go"]
