FROM golang:1.22

WORKDIR /app
COPY . .

COPY cmd/marketflow/config.json .

RUN go build -o marketflow cmd/marketflow/main.go

CMD ["./marketflow"]