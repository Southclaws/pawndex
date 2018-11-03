# -
# Build workspace
# -
FROM golang:1.11 AS compile

RUN apt-get update -y && apt-get install --no-install-recommends -y -q build-essential ca-certificates

WORKDIR /pawndex
ADD . .
RUN make static

# -
# Runtime
# -
FROM scratch

COPY --from=compile /pawndex/pawndex /bin/pawndex
COPY --from=compile /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["pawndex"]
