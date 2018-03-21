FROM golang:alpine
USER root
WORKDIR /

ADD . /

RUN echo "http://mirror1.hs-esslingen.de/pub/Mirrors/alpine/edge/community" >> /etc/apk/repositories
RUN apk update && apk add docker openrc git && rc-update add docker boot 

RUN apk --no-cache update && \
    apk --no-cache add python py-pip py-setuptools ca-certificates curl groff less && \
    pip --no-cache-dir install awscli && \
    rm -rf /var/cache/apk/*

RUN go get github.com/leprosus/golang-slack-notifier
RUN go build /main.go


ARG VCS_REF

LABEL org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url="https://github.com/petertilsen/swarm-scaler"

ENTRYPOINT ["./main"]