FROM aldrinleal/godeb-base:latest

MAINTAINER Aldrin Leal <aldrin@leal.eng.br>

RUN go get github.com/fsouza/go-dockerclient
RUN go get -v github.com/newrelic-forks/dogestry/dogestry

CMD /home/ubuntu/go/dogestry
