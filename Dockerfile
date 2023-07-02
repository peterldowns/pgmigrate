# Build Stage
# Pin to the latest minor version without specifying a patch version so that
# we always deploy security fixes as soon as they are available.
FROM golang:1.20-alpine as builder
RUN apk add build-base

# Have to put our source in the right place for it to build
WORKDIR $GOPATH/src/github.com/peterldowns/pgmigrate

ENV GO111MODULE=on
ENV CGO_ENABLED=1

# Install the dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Build the application
COPY . .

# Put the appropriate build artifacts in a folder for distribution
RUN mkdir -p /dist
RUN go build -o /dist/pgmigrate ./cli

# App Stage
FROM alpine:3.16.3 as app

# Add a non-root user and group with appropriate permissions
# and consistent ids.
RUN addgroup --gid 888 --system pgmigrate && \
    adduser --no-create-home \
            --gecos "" \
            --shell "/bin/ash" \
            --uid 999 \
            --ingroup pgmigrate \
            --system \
            pgmigrate
USER pgmigrate
WORKDIR /app

ARG VERSION=null
ARG COMMIT_SHA=null
ENV COMMIT_SHA=$COMMIT_SHA
LABEL org.opencontainers.image.source="https://github.com/peterldowns/pgmigrate"
LABEL org.opencontainers.image.description="pgmigrate"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"

COPY --from=builder --chown=pgmigrate:pgmigrate /dist /app
CMD ["/app/pgmigrate"]
