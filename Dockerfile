FROM dalf/nghttp2-debian
RUN DEBIAN_FRONTEND=noninteractive apt-get update && apt-get install -y \
  diffutils \
  ssl-cert \
  --no-install-recommends \
  && rm -rf /var/lib/apt/lists/* \
  && make-ssl-cert generate-default-snakeoil --force-overwrite

ADD nghttpx-ingress /
WORKDIR /

CMD ["/nghttpx-ingress"]
