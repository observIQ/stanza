FROM golang:1.17 as stage

ARG plugins_url="https://github.com/observiq/stanza-plugins/releases/latest/download/stanza-plugins.zip"
# arm cross builds do not have these symlinks in palce
RUN \
    ln -s /usr/bin/dpkg-split /usr/sbin/dpkg-split && \
    ln -s /usr/bin/dpkg-deb /usr/sbin/dpkg-deb && \
    ln -s /bin/tar /usr/sbin/tar && \
    ln -s /bin/rm /usr/sbin/rm && \
    echo "resolvconf resolvconf/linkify-resolvconf boolean false" | debconf-set-selections
# unzip is required because tar does not work on arm
RUN apt-get update && apt-get install unzip -y
WORKDIR /stanza/artifacts
RUN curl -fL "${plugins_url}" -o stanza-plugins.zip
RUN unzip stanza-plugins.zip
WORKDIR /stanza
COPY . .
RUN make build
RUN mv "artifacts/stanza_$(go env GOOS)_$(go env GOARCH)" artifacts/stanza


FROM gcr.io/observiq-container-images/stanza-base:v1.1.0

RUN mkdir -p /stanza_home
ENV STANZA_HOME=/stanza_home
RUN echo "pipeline:\n" >> /stanza_home/config.yaml
COPY --from=stage /stanza/artifacts/stanza /stanza_home/stanza
COPY --from=stage /stanza/artifacts/plugins /stanza_home/plugins
ENTRYPOINT /stanza_home/stanza \
  --config /stanza_home/config.yaml \
  --database /stanza_home/stanza.db \
  --plugin_dir /stanza_home/plugins
