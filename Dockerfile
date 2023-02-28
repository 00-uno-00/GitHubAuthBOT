# First stage: build the executable.
FROM golang:buster AS builder

# Create the user and group files that will be used in the running 
# container to run the process as an unprivileged user.
RUN mkdir /user && \
    echo 'GitHubAuthBOT:x:65534:65534:GitHubAuthBOT:/:' > /user/passwd && \
    echo 'GitHubAuthBOT:x:65534:' > /user/group

# Set the Current Working Directory inside the container
WORKDIR $GOPATH/src/github.com/00-uno-00/GitHubAuthBOT

# Import the code from the context.
COPY . .

# Build the executable
RUN go install

# Final stage: the running container.
FROM debian:buster

# Import the user and group files from the first stage.
COPY --from=builder /user/group /user/passwd /etc/

# Copy the built executable
COPY --from=builder /go/bin/GitHubAuthBOT /home/GitHubAuthBOT

# Install dependencies and create home directory
RUN apt update && apt install -y ca-certificates; \ 
    chown -R GitHubAuthBOT /home/GitHubAuthBOT

# Set the workdir
WORKDIR /home/GitHubAuthBOT

# Perform any further action as an unprivileged user.
USER GitHubAuthBOT:GitHubAuthBOT

# Run the compiled binary.
CMD ["./GitHubAuthBOT"]