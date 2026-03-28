# Build (matches go.mod / snap Go on dev host)
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/nexusmods-mcp ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/nexusmods-mcp /nexusmods-mcp
EXPOSE 8080
ENTRYPOINT ["/nexusmods-mcp"]
