#!/bin/sh

echo Installing crossplane ...
kubectl create namespace crossplane-system
helm repo update
helm install crossplane --namespace crossplane-system crossplane-stable/crossplane

echo Sleeping ...
sleep 2

echo Installing provider-kubernetes ...
SA=$(kubectl -n crossplane-system get sa -o name | grep provider-kubernetes | sed -e 's|serviceaccount\/|crossplane-system:|g')
kubectl create clusterrolebinding provider-kubernetes-admin-binding --clusterrole cluster-admin --serviceaccount="${SA}"
kubectl apply -f provider-kubernetes-config.yaml

echo Sleeping ...
sleep 2

echo Applying dummy XRD ...
kubectl apply -f ../database-postgresql/
