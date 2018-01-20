FROM golang AS compile
# just a builder so no need to optimise layers, also makes errors easier to read
RUN apt-get update -y && apt-get install --no-install-recommends -y -q build-essential ca-certificates
RUN go get github.com/golang/dep/cmd/dep
RUN go get github.com/Southclaws/pawndex
WORKDIR /go/src/github.com/Southclaws/pawndex
RUN dep ensure
RUN make static

FROM scratch
COPY --from=compile /go/src/github.com/Southclaws/pawndex/pawndex /bin/pawndex
COPY --from=compile /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["pawndex"]
