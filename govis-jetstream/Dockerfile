FROM golang:1.26.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /demo ./cmd/demo

FROM scratch
COPY --from=build /demo /demo
ENTRYPOINT ["/demo"]
