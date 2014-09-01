FROM aldrinleal/godeb-base:latest

MAINTAINER Aldrin Leal <aldrin@leal.eng.br>

RUN mkdir -p go/src/github.com/fsouza && git clone -b feature/import-export https://github.com/aldrinleal/go-dockerclient go/src/github.com/fsouza/go-dockerclient
RUN go get -v github.com/ingenieux/dogestry/dogestry

CMD /home/ubuntu/go/dogestry
