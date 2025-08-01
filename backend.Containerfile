FROM golang:1.24-bookworm AS build
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

# Create localdb directory for runtime
RUN mkdir -p /localdb && chmod 777 /localdb

FROM gcr.io/distroless/static-debian12


COPY --from=build /bin/backend /bin/backend
COPY --from=build /localdb /localdb

CMD ["/bin/backend"]
