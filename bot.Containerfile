FROM golang:bookworm AS build
LABEL authors="minz1"
WORKDIR /src

ENV GOCACHE=/root/.cache/go-build

COPY go.mod go.sum ./
COPY cmd cmd
COPY internal internal
COPY pkg pkg

RUN --mount=type=cache,target=/root/.cache/go-mod go mod download
RUN --mount=type=cache,target=/root/.cache/go-build GOOS=linux go build -ldflags '-linkmode external -extldflags "-static"' -o /bin/bot ./cmd/bot

FROM alpine
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/bot /bin/bot
CMD ["/bin/bot"]
