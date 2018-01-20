FROM golang AS compile
WORKDIR /go/src/github.com/Southclaws/sampctl
RUN git clone https://github.com/Southclaws/sampctl .
RUN make static

FROM ubuntu
COPY --from=compile /go/src/github.com/Southclaws/sampctl/sampctl /bin/sampctl
RUN mkdir samp && \
    dpkg --add-architecture i386 && \
    apt update && \
    apt install -y g++-multilib git
WORKDIR /samp
ENTRYPOINT ["sampctl"]
