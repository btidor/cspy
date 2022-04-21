FROM golang:alpine AS build

WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -buildvcs=false .

FROM scratch
COPY --from=build /build/cspy /bin/cspy
COPY --from=build /build/static /var/static

USER root
ENTRYPOINT ["cspy"]
