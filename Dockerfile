FROM ubuntu:12.04
MAINTAINER Lachie Cox <lachiec@gmail.com>

RUN apt-get update && apt-get -y install curl \
      git \
      s3cmd=1.1.0* \
      --no-install-recommends
RUN curl -s https://go.googlecode.com/files/go1.2.linux-amd64.tar.gz | tar -v -C /usr/local -xz
ENV	PATH	/usr/local/go/bin:$PATH
ENV	GOPATH	/go:/go/src/github.com/blake-education/dogestry/vendor/go
ADD . /go/src/github.com/blake-education/dogestry

RUN cd /go/src/github.com/blake-education/dogestry && go get && go build

# Setup s3cmd config
RUN	/bin/echo -e '[default]\naccess_key=$AWS_ACCESS_KEY\nsecret_key=$AWS_SECRET_KEY' > /.s3cfg

# Setup s3cmd config
RUN s3cmd --verbose --acl-public put /go/src/github.com/blake-education/dogestry/dogestry s3://ops-data-oregon.blakedev.com/bin/


# TODO push to s3
