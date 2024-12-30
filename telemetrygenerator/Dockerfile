FROM golang:1.23 AS gobuild

ENV PROJECT_ID: ''
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o telemetrygenerator main.go
CMD ["./telemetrygenerator"]
