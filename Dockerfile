# Golang image
FROM golang:1.20

# Working directory
WORKDIR /app

# Copy source code to working directory
COPY . .
