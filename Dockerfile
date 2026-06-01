#1. brings in everything from the golang image, including compiler and other tools
#result is a huge image -> slow deploy and other risks

FROM golang:1.25-alpine AS builder
WORKDIR /app

#copy dependencies 
COPY go.mod go.sum ./

#download dependencies
RUN go mod download

#copy source code
COPY . .

#build the application and name
RUN go build -o url_shortener .


#2. use smaller image using alpine bring only the compiled binary and necessary libraries 
#from previous stage grab only compiled binary in the url_shortner and copy to new image
#result is a smaller image -> faster deploy and less attack surface

FROM alpine:latest
WORKDIR /app

#from the builder stage copy the compiled binary to the new image
COPY --from=builder /app/url_shortener .

#when docker run command is executed, it will run the url_shortener binary
CMD ["./url_shortener"]