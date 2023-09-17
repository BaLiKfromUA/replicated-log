FROM golang:latest

RUN mkdir /app
ADD . /app/
WORKDIR /app

RUN go mod tidy
RUN go build -o main ./cmd/
CMD ["/app/main"]