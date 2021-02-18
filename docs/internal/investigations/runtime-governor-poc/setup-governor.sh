#!/usr/bin/env bash
NETWORK="cpnet"

k3d cluster create governor --network $NETWORK
sleep 10s

kubectl --context=k3d-governor create ns redis1-system 
kubectl --context=k3d-governor create ns redis2-system 
helm --kube-context=k3d-governor repo add bitnami https://charts.bitnami.com/bitnami
helm --kube-context=k3d-governor upgrade -i redis bitnami/redis -n redis1-system
helm --kube-context=k3d-governor upgrade -i redis bitnami/redis -n redis2-system

CP_IPADDR=$(docker inspect k3d-governor-server-0 --format='{{json .NetworkSettings.Networks.cpnet.IPAddress}}')
REDIS1_PASSWORD=$(kubectl --context=k3d-governor get secret -n redis1-system redis -ojsonpath='{.data.redis-password}' | base64 -d)
REDIS2_PASSWORD=$(kubectl --context=k3d-governor get secret -n redis2-system redis -ojsonpath='{.data.redis-password}' | base64 -d)

kubectl --context=k3d-governor create ns cp-poc
helm --kube-context=k3d-governor upgrade -i cp-governor -n cp-poc ./governor/chart \
  --set redis1.host=${CP_IPADDR//\"}:30000 --set redis1.password=$REDIS1_PASSWORD \
  --set redis2.host=${CP_IPADDR//\"}:31000 --set redis2.password=$REDIS2_PASSWORD

kubectl --context=k3d-governor apply -f ./governor/redis-svc.yaml
