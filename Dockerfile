# worldd for a small VPS (deploy/README.md). CGO is needed for H3, so the
# builder has gcc; the runtime image only needs libc.
FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY third_party ./third_party
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /out/worldd ./server/cmd/worldd \
 && CGO_ENABLED=1 go build -o /out/worldq ./server/cmd/worldq

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /strata
COPY --from=build /out/worldd /out/worldq /usr/local/bin/
COPY rules ./rules
# The snapshot is data, not part of the image: mount it at /strata/snapshot.
EXPOSE 8080
ENTRYPOINT ["worldd", "-snapshot", "/strata/snapshot", "-rules", "/strata/rules", "-addr", ":8080"]
