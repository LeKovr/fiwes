# Dockerfile
# based on https://medium.com/@pierreprinetti/the-go-1-11-dockerfile-a3218319d191

ARG GO_VERSION="1.12.4"

# Don't use alpine decause we run `go test -race`
# See https://github.com/golang/go/issues/14481
FROM golang:${GO_VERSION} as builder

RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group

# Installing golangci-lint
RUN curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${GOPATH}/bin" v1.16.0

# golint
RUN go get -u golang.org/x/lint/golint

# app
WORKDIR /app

# Cached layer
COPY ./go.mod ./go.sum ./
RUN go mod download

# Sources dependent layer
COPY ./ ./
RUN make build-standalone

FROM scratch AS final

LABEL maintainer="lekovr+docker@gmail.com" \
      org.label-schema.description="File web storage server" \
      org.label-schema.schema-version="1.0" \
      org.label-schema.url="https://github.com/LeKovr/fiwes" \
      org.label-schema.vcs-url="https://github.com/LeKovr/fiwes"

# TODO: Found a way to set this from `git describe --tags`
#      org.label-schema.version="${APP_VERSION}"
# ARG APP_VERSION

COPY --from=builder /user/group /user/passwd /etc/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/fiwes /fiwes

# Extend golang content type support
COPY --from=builder /etc/mime.types /etc/mime.types

VOLUME /data

WORKDIR /
# Adding static as files allows changing them later
COPY assets assets

# Server port to listen
ENV PORT 8080

# Expose the server TCP port
EXPOSE ${PORT}

# Perform any further action as an unprivileged user.
# Custom entrypoint will be needed for store dir permissions set
#USER nobody:nobody

# Run the compiled binary.
ENTRYPOINT ["/fiwes"]
