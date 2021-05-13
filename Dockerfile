FROM golang:1.16-buster AS build
ENV GOPROXY=https://proxy.golang.org
WORKDIR /go/src/vax-notifier
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/vax-notifier .

FROM registry.access.redhat.com/ubi7/ubi
RUN yum update -y && yum install -y ca-certificates && mkdir /app
COPY --from=build /go/bin/vax-notifier /app/
COPY --from=build /go/src/vax-notifier/config.yaml /app/
WORKDIR /app
USER root
ENTRYPOINT ["./vax-notifier"]