# Use an official Go runtime as a parent image
FROM golang:1.21.5

# Set the working directory in the container
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . .

# Download Go modules
RUN go mod download

# Build the Go app
RUN go build -o main .

# Run the app when the container launches
CMD ["./main"]
