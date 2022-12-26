FROM golang:1.18.1-alpine3.15 as builder

MAINTAINER brant

WORKDIR /go/multitude_rabbitmq_exporter

COPY . /go/multitude_rabbitmq_exporter

ENV GOPROXY https://goproxy.cn,direct

RUN GOOS=linux GOFLAGS=-buildvcs=false GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s"


# runner
FROM alpine AS runner

COPY --from=builder /go/multitude_rabbitmq_exporter/rabbitmq_exporter /rabbitmq_exporter

# 将时区设置为东八区
#RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
#    && apk update \
#    && apk upgrade \
#    && apk add --no-cache tzdata \
#    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime  \
#    && echo Asia/Shanghai > /etc/timezone \
#    && apk del tzdata \

WORKDIR /

ENTRYPOINT [ "/rabbitmq_exporter"]









#FROM alpine AS builder
#
## Install the Certificate-Authority certificates for the app to be able to make
## calls to HTTPS endpoints.
## Git is required for fetching the dependencies.
#RUN apk add --no-cache ca-certificates
#
## Final stage: the running container.
#FROM scratch AS final
#
## Add maintainer label in case somebody has questions.
#LABEL maintainer="Kris.Budde@gmail.com"
#
## Import the Certificate-Authority certificates for enabling HTTPS.
#COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
#
## Import the compiled executable from the first stage.
#COPY rabbitmq_exporter /rabbitmq_exporter
#
## Declare the port on which the webserver will be exposed.
## As we're going to run the executable as an unprivileged user, we can't bind
## to ports below 1024.
#EXPOSE 9419
#
## Perform any further action as an unprivileged user.
#USER 65535:65535
#
## Check if exporter is alive; 10 retries gives prometheus some time to retrieve bad data (5 minutes)
#HEALTHCHECK --retries=10 CMD ["/rabbitmq_exporter", "-check-url", "http://localhost:9419/health"]
#
## Run the compiled binary.
#ENTRYPOINT ["/rabbitmq_exporter"]
