FROM google/golang

RUN go get github.com/fsouza/go-dockerclient
RUN go get -v github.com/newrelic-forks/dogestry/dogestry

CMD /home/ubuntu/go/dogestry
