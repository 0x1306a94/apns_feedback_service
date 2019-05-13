FROM golang:1.11-alpine
MAINTAINER 0x1306a94@gmail.com

WORKDIR /go/src

COPY ./app ./apns_feedback_service/app

RUN cd /go/src/apns_feedback_service/app && \
    go install . && \
    which app

CMD ["app", "--config", "/go/src/apns_feedback_service/app/config.json"]