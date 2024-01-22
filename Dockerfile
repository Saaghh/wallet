FROM golang:latest

WORKDIR /app

COPY . .

RUN make build

CMD ["./bin/apiserver"]