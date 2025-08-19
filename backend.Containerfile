FROM golang:1.25-bookworm AS build
LABEL authors="minz1"
WORKDIR /src

# Set up build environment for performance
ENV GOCACHE=/root/.cache/go-build

COPY go.mod go.sum ./
COPY cmd cmd
COPY internal internal
COPY pkg pkg

RUN --mount=type=cache,target=/go/pkg/mod go mod download
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=linux go build \
    -ldflags='-s -w -linkmode external -extldflags "-static"' \
    -gcflags="-l=4" \
    -o /bin/backend ./cmd/backend

FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=build /bin/backend /bin/backend

CMD ["/bin/backend"]
