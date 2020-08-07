#!/usr/bin/env bash
CP_IPADDR=$(docker inspect k3d-governor-server-0 --format='{{json .NetworkSettings.Networks.cpnet.IPAddress}}')

helm --kube-context=k3d-runtime1 upgrade -i cp-agent ./runtime/chart -n cp-poc --set agent.governorURL=http://${CP_IPADDR//\"}
