FROM ubuntu:bionic

RUN mkdir -p /bplogagent_home/plugins
ENV BPLOGAGENT_HOME=/bplogagent_home
RUN echo "pipeline:\n" >> /bplogagent_home/config.yaml

COPY ./artifacts/bplogagent_linux_amd64 /bplogagent_home/bplogagent
ENTRYPOINT /bplogagent_home/bplogagent \
  --config /bplogagent_home/config.yaml \
  --database /bplogagent_home/bplogagent.db \
  --plugin_dir /bplogagent_home/plugins
