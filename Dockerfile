FROM golang:1.21

WORKDIR /app

COPY . .

RUN go mod download
RUN GOOS=linux GOARCH=amd64 go build -o /api ./cmd/api

EXPOSE 8080

CMD ["/api"]
