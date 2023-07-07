# Build Stage
# Pin to the latest minor version without specifying a patch version so that
# we always deploy security fixes as soon as they are available.
FROM golang:1.20-alpine as builder
RUN apk add build-base git

# Have to put our source in the right place for it to build
WORKDIR $GOPATH/src/github.com/peterldowns/pgmigrate

ENV GO111MODULE=on
ENV CGO_ENABLED=1

# Install the dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download
RUN mkdir -p cmd/pgmigrate
COPY cmd/pgmigrate/go.mod cmd/pgmigrate
COPY cmd/pgmigrate/go.sum cmd/pgmigrate
RUN cd cmd/pgmigrate && go mod download

# Build the application
COPY . .

# Put the appropriate build artifacts in a folder for distribution
RUN mkdir -p /dist

ARG VERSION=unknown
ARG COMMIT_SHA=unknown

ENV PGM_VERSION=$VERSION
ENV PGM_COMMIT_SHA=$COMMIT_SHA

RUN go build \
  -ldflags "-X github.com/peterldowns/pgmigrate/cmd/pgmigrate/shared.Version=${PGM_VERSION} -X github.com/peterldowns/pgmigrate/cmd/pgmigrate/shared.Commit=${PGM_COMMIT_SHA}" \
  -o /dist/pgmigrate \
  ./cmd/pgmigrate

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

ARG VERSION=unknown
ARG COMMIT_SHA=unknown

ENV PGM_VERSION=$VERSION
ENV PGM_COMMIT_SHA=$COMMIT_SHA

LABEL org.opencontainers.image.source="https://github.com/peterldowns/pgmigrate"
LABEL org.opencontainers.image.description="pgmigrate"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${PGM_VERSION}"
LABEL org.opencontainers.image.revision="${PGM_COMMIT_SHA}"

COPY --from=builder --chown=pgmigrate:pgmigrate /dist /app
ENV PATH="/app:$PATH"
CMD ["/app/pgmigrate"]
