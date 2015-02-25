FROM google/golang

RUN go get github.com/fsouza/go-dockerclient
RUN go get -v github.com/dogestry/dogestry

CMD /home/ubuntu/go/dogestry
