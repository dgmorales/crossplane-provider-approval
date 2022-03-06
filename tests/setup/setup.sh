#!/bin/sh

echo Installing crossplane ...
kubectl create namespace crossplane-system
helm repo update
helm install crossplane --namespace crossplane-system crossplane-stable/crossplane

echo Sleeping ...
sleep 2

echo Installing provider-kubernetes ...
kubectl crossplane install provider crossplane/provider-kubernetes:main
SA=$(kubectl -n crossplane-system get sa -o name | grep provider-kubernetes | sed -e 's|serviceaccount\/|crossplane-system:|g')
kubectl create clusterrolebinding provider-kubernetes-admin-binding --clusterrole cluster-admin --serviceaccount="${SA}"
kubectl apply -f tests/setup/provider-kubernetes-config.yaml

# add this to cluster-admin clusterrolebinding
# - kind: ServiceAccount
#   name: crossplane
#   namespace: crossplane-system

echo Sleeping ...
sleep 2

echo Applying dummy XRD ...
kubectl apply -f ../database-postgresql/
