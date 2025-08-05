FROM golang:1.22

WORKDIR /app
COPY . .

RUN go build -o datagen cmd/datagen/main.go

CMD ["./datagen"]