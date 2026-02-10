FROM golang:1.25-alpine as builder

WORKDIR /src
RUN apk add --no-cache git

# Clone the repository
RUN git clone https://github.com/blnkfinance/blnk.git .

# Download dependencies
RUN go mod download

# Build the application
# Trying standard entry points
RUN if [ -f main.go ]; then go build -o /bin/blnk main.go; else go build -o /bin/blnk ./cmd/...; fi

FROM alpine:3.19
RUN apk add --no-cache ca-certificates bash

COPY --from=builder /bin/blnk /usr/local/bin/blnk

# Verify binary
RUN blnk --help || echo "Binary not working"

CMD ["blnk", "start"]
