FROM golang:1.26.9 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /box-view ./cmd/box-view

FROM scratch
COPY --from=build /box-view /box-view
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/box-view"]
CMD ["-addr", "0.0.0.0:8080", "-db", "/data/model.db"]
