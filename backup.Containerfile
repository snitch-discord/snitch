FROM golang:1.25-bookworm AS build
LABEL authors="sillyvan"
WORKDIR /src

# Set up build environment for performance
ENV GOCACHE=/root/.cache/go-build

COPY go.mod go.sum ./
COPY cmd cmd
COPY internal internal
COPY pkg pkg

RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-s -w' \
    -gcflags="-l=4" \
    -o /bin/backup ./cmd/backup

FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=build /bin/backup /bin/backup

CMD ["/bin/backup"]

