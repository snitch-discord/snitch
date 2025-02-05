FROM golang:bookworm AS build
LABEL authors="minz1"
WORKDIR /src

ENV GOCACHE=/root/.cache/go-build

COPY go.mod go.sum ./
COPY cmd cmd
COPY internal internal
COPY pkg pkg

RUN --mount=type=cache,target=/root/.cache/go-mod go mod download
RUN --mount=type=cache,target=/root/.cache/go-build GOOS=linux go build -ldflags '-linkmode external -extldflags "-static"' -o /bin/backend ./cmd/backend

FROM debian:stable-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /localdb && \
    chmod 777 /localdb

COPY --from=build /bin/backend /bin/backend

CMD ["/bin/backend"]
