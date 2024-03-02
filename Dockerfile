# Use an official Go runtime as a parent image
FROM docker.io/golang:latest as builder

# Set the working directory in the container
WORKDIR /go/src/github.com/TedSinger/blurbcloud

# Clone the repository
RUN git clone https://github.com/TedSinger/blurbcloud.git .

# Install dependencies and build the project using the rakefile task
RUN go get -d -v ./...
RUN go install -v ./...
RUN go build -o blurbcloud .

# Copy the necessary files as per the rakefile task
RUN cp queries.sql /go/bin
RUN cp -r static /go/bin

# Use a smaller image to run the application
FROM debian:bookworm-slim  

WORKDIR /root/

# Copy the binary and other files from the builder image
COPY --from=builder /go/bin/blurbcloud .
COPY --from=builder /go/bin/queries.sql .
COPY --from=builder /go/bin/static ./static

# Make the blurbcloud binary executable
RUN chmod +x ./blurbcloud

# The command to run the application
CMD ["./blurbcloud", "-db", "/omb-local/blurbcloud/blurbs.db"]
