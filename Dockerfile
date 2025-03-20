FROM golang:1.23.4-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY *.go ./

RUN go mod download
RUN go build -o main ./...

CMD ["./main"]
