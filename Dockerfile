FROM golang:1.21.7-alpine3.19 as build

WORKDIR /build

# Copy Source Code
COPY . .

# Install Plugin
RUN go build -o /usr/local/bin/plugin main.go
RUN chmod +x /usr/local/bin/plugin

FROM hashicorp/terraform:1.6.5

WORKDIR /work

COPY --from=build /usr/local/bin/plugin /usr/local/bin/plugin

ENTRYPOINT [ "/usr/local/bin/plugin" ]
