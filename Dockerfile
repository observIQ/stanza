FROM golang:1.16 as stage

WORKDIR /stanza
COPY . .
RUN rm -rf artifacts/*
RUN make build
WORKDIR /stanza/artifacts
# hack: "mv stanza_* stanza" gives an error because mv does not like '_*'
RUN for f in stanza_*; do mv "$f" stanza; done

FROM gcr.io/observiq-container-images/stanza-base:v1.1.0

RUN mkdir -p /stanza_home
ENV STANZA_HOME=/stanza_home
RUN echo "pipeline:\n" >> /stanza_home/config.yaml

COPY --from=stage /stanza/artifacts/stanza /stanza_home/stanza
COPY ./artifacts/stanza-plugins.tar.gz /tmp/stanza-plugins.tar.gz
RUN tar -zxvf /tmp/stanza-plugins.tar.gz -C /stanza_home/
ENTRYPOINT /stanza_home/stanza \
  --config /stanza_home/config.yaml \
  --database /stanza_home/stanza.db \
  --plugin_dir /stanza_home/plugins
