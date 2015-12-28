FROM golang:1.5.2

MAINTAINER yvonnick.esnault@corp.ovh.com

RUN mkdir -p /go/src/stash.ovh.net/textandtags/tatsamon
WORKDIR /go/src/stash.ovh.net/textandtags/tatsamon

# this will ideally be built by the ONBUILD below ;)
CMD ["go-wrapper", "run"]

COPY . /go/src/stash.ovh.net/textandtags/tatsamon
RUN go-wrapper download
RUN go-wrapper install
