#!/bin/bash

BPLOG_ROOT="$GOPATH/src/github.com/bluemedora/bplogagent"

PROJECT="bindplane-agent-dev-0"
ZONE="us-central1-a"
INSTANCE="rhel6"

if ! [ -x "$(command -v gcloud)" ]; then
  echo 'Error: gcloud is not installed.' >&2
  exit 1
fi

if ! [ -x "$(command -v dot)" ]; then
  echo 'Error: dot is not installed.' >&2
  exit 1
fi

echo "Building log agent"
GOOS=linux go build -o /tmp/bplogagent $BPLOG_ROOT/cmd

echo "Building benchmark tool"
GOOS=linux go build -o /tmp/logbench github.com/awslabs/amazon-log-agent-benchmark-tool/cmd/logbench/

echo "Setting up benchmark run on $PROJECT $ZONE $INSTANCE"
gcloud beta compute --project "$PROJECT" ssh --zone "$ZONE" "$INSTANCE" -- 'rm -rf ~/benchmark && mkdir ~/benchmark'
gcloud beta compute --project "$PROJECT" scp --zone "$ZONE" /tmp/bplogagent "$INSTANCE:~/benchmark"
gcloud beta compute --project "$PROJECT" scp --zone "$ZONE" /tmp/logbench "$INSTANCE:~/benchmark/LogBench"
gcloud beta compute --project "$PROJECT" scp --zone "$ZONE" $BPLOG_ROOT/scripts/benchmark/config.yaml "$INSTANCE:~/benchmark/config.yaml"
gcloud beta compute --project "$PROJECT" ssh --zone "$ZONE" "$INSTANCE" -- 'chmod -R 777 ~/benchmark'

echo "Running single-file benchmark"
gcloud beta compute --project "$PROJECT" ssh --zone "$ZONE" "$INSTANCE" -- \
  'set -m
  ~/benchmark/LogBench -log stream.log -rate 100 -t 60s -r 30s -f 2s ~/benchmark/bplogagent --config ~/benchmark/config.yaml > ~/benchmark/output1 2>&1 &
  sleep 10;
  curl http://localhost:6060/debug/pprof/profile?seconds=30 > ~/benchmark/profile1 ; 
  fg ; '

echo "Running 20-file benchmark"
gcloud beta compute --project "$PROJECT" ssh --zone "$ZONE" "$INSTANCE" -- \
  'set -m
  ~/benchmark/LogBench -log $(echo stream{1..20}.log | tr " " ,) -rate 100 -t 60s -r 30s -f 2s ~/benchmark/bplogagent --config ~/benchmark/config.yaml > ~/benchmark/output20 2>&1 &
  sleep 10;
  curl http://localhost:6060/debug/pprof/profile?seconds=30 > ~/benchmark/profile20 ; 
  fg ; '

output_dir="$BPLOG_ROOT/tmp/$(date +%Y-%m-%d_%H-%M-%S)"
mkdir -p $output_dir
echo "Results will be located in $output_dir"

echo "Retrieving results"
gcloud beta compute --project "$PROJECT" scp --zone "$ZONE" "$INSTANCE:~/benchmark/output*" "$output_dir"
gcloud beta compute --project "$PROJECT" scp --zone "$ZONE" "$INSTANCE:~/benchmark/profile*" "$output_dir"

echo "Opening profiles"
go tool pprof -http localhost:6001 $output_dir/profile1 &
go tool pprof -http localhost:6020 $output_dir/profile20 &