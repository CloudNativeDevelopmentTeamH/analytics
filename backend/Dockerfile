FROM golang:1.24-alpine AS builder

LABEL maintainer="Vic Shóstak <vic@shostak.dev> (https://shostak.dev/)"

# Move to working directory (/build).
WORKDIR /build

# Copy and download dependency using go mod.
COPY go.mod go.sum ./
RUN go mod download

# Copy the code into the container.
COPY . .

# Set necessary environment variables needed for our image and build the API server.
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o apiserver .

FROM scratch

# Copy binary and runtime config into scratch image.
COPY --from=builder ["/build/apiserver", "/apiserver"]
COPY --from=builder /build/config /config

EXPOSE 8080 9090

# Command to run when starting the container.
ENTRYPOINT ["/apiserver"]
