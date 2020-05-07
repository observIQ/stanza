#!/bin/bash

echo "Building log agent"
GOOS=linux go build -o /tmp/bplogagent ../cmd

echo "Copying log agent"
gcloud beta compute --project "***REMOVED***" ssh --zone "us-east1-b" 'instance-1' -- 'rm -rf /tmp/benchmark && mkdir /tmp/benchmark'
gcloud beta compute --project "***REMOVED***" scp --zone "us-east1-b" /tmp/bplogagent 'instance-1:/tmp/benchmark'

echo "Downloading dependencies"
gcloud beta compute --project "***REMOVED***" ssh --zone "us-east1-b" 'instance-1' -- \
  'gsutil cp gs://bplogagent-logbench/LogBench /tmp/benchmark/LogBench && chmod +x /tmp/benchmark/LogBench'
gcloud beta compute --project "***REMOVED***" ssh --zone "us-east1-b" 'instance-1' -- \
  'gsutil cp gs://bplogagent-logbench/config.yaml /tmp/benchmark/config.yaml'

echo "Running single-file benchmark"
gcloud beta compute --project "***REMOVED***" ssh --zone "us-east1-b" 'instance-1' -- \
  'set -m ;
  cd /tmp/benchmark ;
  /tmp/benchmark/LogBench -log stream.log -rate 100 -t 60s -r 30s -f 2s /tmp/benchmark/bplogagent --config /tmp/benchmark/config.yaml > output1 &
  sleep 10;
  curl http://localhost:6060/debug/pprof/profile?seconds=30 > /tmp/benchmark/profile1 ;
  fg ; '

echo "Running 20-file benchmark"
gcloud beta compute --project "***REMOVED***" ssh --zone "us-east1-b" 'instance-1' -- \
  'set -m ;
  cd /tmp/benchmark ;
  /tmp/benchmark/LogBench -log $(echo stream{1..20}.log | tr " " ,) -rate 100 -t 60s -r 30s -f 2s /tmp/benchmark/bplogagent --config /tmp/benchmark/config.yaml > output20 &
  sleep 10;
  curl http://localhost:6060/debug/pprof/profile?seconds=30 > /tmp/benchmark/profile20 ;
  fg ; '

echo "Retrieving results"
output_dir="/tmp/benchmark/$(date +%Y-%m-%d_%H-%M-%S)"
mkdir -p $output_dir
gcloud beta compute --project "***REMOVED***" scp --zone "us-east1-b" 'instance-1:/tmp/benchmark/output*' "$output_dir"
gcloud beta compute --project "***REMOVED***" scp --zone "us-east1-b" 'instance-1:/tmp/benchmark/profile*' "$output_dir"
echo "Results are located in $output_dir"
