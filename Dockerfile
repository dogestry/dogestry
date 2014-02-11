FROM ubuntu:13.10
MAINTAINER Lachie Cox <lachiec@gmail.com>

RUN apt-get update && \
      apt-get -y install \
      curl \
      git \
      ca-certificates \
      --no-install-recommends

RUN curl -s https://go.googlecode.com/files/go1.2.linux-amd64.tar.gz | tar -v -C /usr/local -xz
ENV	PATH	/usr/local/go/bin:$PATH
ENV	GOPATH	/go:/go/src/github.com/blake-education/dogestry/vendor/go
ADD . /go/src/github.com/blake-education/dogestry

RUN cd /go/src/github.com/blake-education/dogestry && \
    go get && \
    go build && \
    cp dogestry /dogestry
