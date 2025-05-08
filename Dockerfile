FROM golang:1.24.1-alpine

# Install required packages
RUN apk add --no-cache git protobuf make

# Install protoc-gen-go
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

ENV PATH="/go/bin:$PATH"

WORKDIR /app

# Copy source code
COPY . .

# Generate protobuf files
RUN protoc --go_out=. --go_opt=paths=source_relative proto/*.proto

# Build via Makefile
RUN make build

EXPOSE 3998
CMD make run
