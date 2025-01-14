FROM golang:1.23.3-alpine3.20 AS build
WORKDIR /
COPY . .
RUN go mod download && \
    CGO_ENABLED=0 go build -o coredns ./cmd/coredns.go

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /coredns /coredns
USER nonroot:nonroot
WORKDIR /
EXPOSE 53 53/udp
ENTRYPOINT ["/coredns"]
