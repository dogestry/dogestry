FROM aldrinleal/godeb-base:latest

MAINTAINER Aldrin Leal <aldrin@leal.eng.br>

RUN go get github.com/ingenieux/dogestry/dogestry

CMD /home/ubuntu/go/dogestry