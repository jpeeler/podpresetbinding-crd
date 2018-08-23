# PodPresetBindings CRD

The source in the repository contains a PodPresetBinding resource that's implemented
via a CRD built with Kubebuilder. The code here depends on installing and running
the [PodPreset CRD](https://github.com/jpeeler/podpreset-crd) resource as well.
The goal of this CRD is to provide an easy method for synchronizing a given
deployment with binding credentials that are eventually made ready by
[service-catalog](https://github.com/kubernetes-incubator/service-catalog).

## Getting started

In order to deploy the PodPresetBinding resource into your cluster, do the following:

1. Deploy Kubernetes.

1. Deploy PodPreset CRD.

1. Install [kustomize](https://github.com/kubernetes-sigs/kustomize) (which
   was also mentioned in the instructions of the other repository).

1. Run CRD, choose to execute inside or outside the cluster.

   Inside cluster:

   ```shell
   make docker-build
   make deploy
   ```

   Outside cluster (for debugging/development):

   ```shell
   make install
   make run
   ```

1. Apply desired pod preset bindings as needed, example given below.

## Example usage

```shell
kubectl create -f config/samples/apod2-presetbinding.yaml
kubectl create -f config/samples/apod2-deployment.yaml
```

## Additional information

### Service Catalog integration

As mentioned before, the code here works with service catalog and the podpreset
CRD code. The reconcile loop in this repository is responsible for handling
creation and updates of pod preset resources based on the spec of a
podpresetbinding. The podpreset CRD reconcile loop handles restarting
deployments for which pod presets have mutated.

### WARNING

Until this code is moved to an official Kubernetes namespaced repository, this
repo may undergo force pushes to add additional necessary changes.
