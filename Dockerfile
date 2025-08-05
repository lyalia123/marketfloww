FROM golang:1.22

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -o marketflow ./cmd/marketflow/main.go

EXPOSE 8080

CMD ["./marketflow"]
