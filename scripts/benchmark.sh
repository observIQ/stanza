#!/bin/bash

BPLOG_ROOT="$GOPATH/src/github.com/bluemedora/bplogagent"

run_time=$(date +%Y-%m-%d-%H-%M-%S)

PROJECT="bindplane-agent-dev-0"
ZONE="us-central1-a"
INSTANCE="rhel6-$run_time"

if ! [ -x "$(command -v gcloud)" ]; then
  echo 'Error: gcloud is not installed.' >&2
  exit 1
fi

if ! [ -x "$(command -v dot)" ]; then
  echo 'Error: dot is not installed.' >&2
  exit 1
fi

echo "Creating Instance: $INSTANCE [$PROJECT] [$ZONE]"
gcloud beta compute instances create --verbosity=error \
  --project=$PROJECT --zone=$ZONE $INSTANCE --preemptible \
  --image=rhel-6-v20200402 --image-project=rhel-cloud \
  --machine-type=n1-standard-1 --boot-disk-size=200GB > /dev/null

echo "Waiting for instance to be ready"
until gcloud beta compute ssh --verbosity=critical --project $PROJECT --zone $ZONE $INSTANCE --ssh-flag="-o LogLevel=QUIET" -- 'echo "Ready"'; do
  echo "VM not ready. Waiting..."  
done

echo "Building log agent"
GOOS=linux go build -o /tmp/bplogagent $BPLOG_ROOT/cmd

echo "Building benchmark tool"
GOOS=linux go build -o /tmp/logbench github.com/bluemedora/amazon-log-agent-benchmark-tool/cmd/logbench/

echo "Setting up benchmark test"
gcloud beta compute ssh --project $PROJECT --zone $ZONE $INSTANCE --ssh-flag="-o LogLevel=QUIET" -- 'rm -rf ~/benchmark && mkdir ~/benchmark' > /dev/null
gcloud beta compute scp --project $PROJECT --zone $ZONE /tmp/bplogagent $INSTANCE:~/benchmark > /dev/null
gcloud beta compute scp --project $PROJECT --zone $ZONE /tmp/logbench $INSTANCE:~/benchmark/LogBench > /dev/null
gcloud beta compute scp --project $PROJECT --zone $ZONE $BPLOG_ROOT/scripts/benchmark/config.yaml $INSTANCE:~/benchmark/config.yaml > /dev/null
gcloud beta compute ssh --project $PROJECT --zone $ZONE $INSTANCE --ssh-flag="-o LogLevel=QUIET" -- 'chmod -R 777 ~/benchmark' > /dev/null

echo "Running single-file benchmark (60 seconds per test)"
gcloud beta compute ssh --project $PROJECT --zone $ZONE $INSTANCE --ssh-flag="-o LogLevel=QUIET" -- \
  'set -m
  ~/benchmark/LogBench -log stream.log -rate 100,1k,10k -t 60s -r 30s -f 2s -out ~/benchmark/results1.json ~/benchmark/bplogagent --config ~/benchmark/config.yaml > ~/benchmark/notes1 2>&1 &
  sleep 10;
  curl http://localhost:6060/debug/pprof/profile?seconds=160 > ~/benchmark/profile1 ; 
  fg ; ' > /dev/null

echo "Running 10-file benchmark (60 seconds per test)"
gcloud beta compute ssh --project $PROJECT --zone $ZONE $INSTANCE --ssh-flag="-o LogLevel=QUIET" -- \
  'set -m
  ~/benchmark/LogBench -log $(echo stream{1..10}.log | tr " " ,) -rate 100,1k,10k -t 60s -r 30s -f 2s -out ~/benchmark/results10.json ~/benchmark/bplogagent --config ~/benchmark/config.yaml > ~/benchmark/notes10 2>&1 &
  sleep 10;
  curl http://localhost:6060/debug/pprof/profile?seconds=160 > ~/benchmark/profile10 ; 
  fg ; ' > /dev/null

output_dir="$BPLOG_ROOT/tmp/$run_time"
mkdir -p $output_dir

echo "Retrieving results"
gcloud beta compute scp --project $PROJECT --zone $ZONE $INSTANCE:~/benchmark/results* $output_dir > /dev/null
gcloud beta compute scp --project $PROJECT --zone $ZONE $INSTANCE:~/benchmark/notes* $output_dir > /dev/null
gcloud beta compute scp --project $PROJECT --zone $ZONE $INSTANCE:~/benchmark/profile* $output_dir > /dev/null

echo "Cleaning up instance"
gcloud beta compute instances delete --quiet --project $PROJECT --zone=$ZONE $INSTANCE

echo
echo "Result files"
echo "  $output_dir/results1.json"
echo "  $output_dir/results10.json"

echo
echo "stdout"
echo "  $output_dir/notes1"
echo "  $output_dir/notes10"

echo
echo "Profiles can be accessed with the following commands"
echo "  go tool pprof -http localhost:6001 $output_dir/profile1"
echo "  go tool pprof -http localhost:6010 $output_dir/profile10"