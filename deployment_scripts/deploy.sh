#!/usr/bin/env bash
kubectl delete deployment twitter-service
kubectl delete deployment bbc-service
kubectl delete deployment analysis-service
kubectl delete deployment redis
kubectl delete deployment redis-slave
kubectl delete deployment client

kubectl delete services bbc-service
kubectl delete services twitter-service
kubectl delete services analysis-service
kubectl delete services redis
kubectl delete services redis-slave
kubectl delete services client

kubectl create -f twitter.yaml,bbc.yaml,analysis.yaml,client.yaml,redis-master.yaml,redis-slave.yaml
