#!/bin/bash

set -e

#Provide default value of APP_NS
APP_NS=${APP_NS:="default"}
IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY:="Always"}
EXPERIMENT_IMAGE=${EXPERIMENT_IMAGE:="litmuschaos/go-runner"}
EXPERIMENT_IMAGE_TAG=${EXPERIMENT_IMAGE_TAG:="latest"}
PUMBA_LIB=${PUMBA_LIB:="pumba"}

#Add chaos helm repository
helm repo add k8s-chaos https://litmuschaos.github.io/litmus-helm/
helm repo list
helm search repo k8s-chaos

#Install the kubernetes chaos experiments
helm install k8s k8s-chaos/kubernetes-chaos --set image.litmusGO.pullPolicy=${IMAGE_PULL_POLICY} \
--set image.litmusGO.repository=${EXPERIMENT_IMAGE} --set image.litmusGO.tag=${EXPERIMENT_IMAGE_TAG} \
--set image.pumba.repository=${EXPERIMENT_IMAGE} --set image.pumba.libName=${PUMBA_LIB} \
--set image.litmusLIBImage.repository=${EXPERIMENT_IMAGE} --set image.litmusLIBImage.tag=${EXPERIMENT_IMAGE_TAG} \
--set image.pumba.tag=${EXPERIMENT_IMAGE_TAG}  --namespace=${APP_NS}

#Checking the installation 
kubectl get chaosexperiments -n ${APP_NS}
