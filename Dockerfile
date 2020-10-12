FROM golang:1.15.2-buster

# Install protocol buffers
RUN apt update
RUN apt install -y protobuf-compiler
RUN go get google.golang.org/protobuf/cmd/protoc-gen-go
RUN go get google.golang.org/grpc/cmd/protoc-gen-go-grpc
ENV PATH="/go/bin:${PATH}"

# Install dependencies
ADD . /code/
WORKDIR /code/
RUN go install

# Compile protocol buffers and gRPC service
RUN make install

# Set endpoint
ENTRYPOINT [ "real_feed" ]