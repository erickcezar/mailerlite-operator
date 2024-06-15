# mailerlite-operator

The `mailerlite-operator` is a Kubernetes operator designed to manage and automate the sending of transactional emails using multiple providers, such as MailerSend and Mailgun. This operator provides custom resource definitions (CRDs) for configuring email sender settings and defining email messages. With cross-namespace capabilities, it monitors and responds to changes in email configurations and triggers email sending processes accordingly, updating the status of email resources to reflect delivery outcomes.

## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

For this demo, We are using minikube. For more details how to install it, click [here](https://minikube.sigs.k8s.io/docs/)

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
docker-build docker-push IMG=<some-registry>/mailerlite-operator:tag
```

For this case, We don't need to run push as we are using local docker.
But we need to run this command:

```
eval $(minikube docker-env)
```

So minikube can push images locally. Remember to turn off the imagePullPolicy:Always (use imagePullPolicy:IfNotPresent or imagePullPolicy:Never) in your yaml file. Otherwise Kubernetes won’t use your locally build image and it will pull from the network.

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/mailerlite-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/mailerlite-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/mailerlite-operator/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## References

[mailsender api doc](https://developers.mailersend.com/api/v1/email.html)  
[mailgun api doc](
https://documentation.mailgun.com/docs/mailgun/api-reference/openapi-final/tag/Messages/#tag/Messages/operation/httpapi.(*apiHandler).handler-fm-18)  
[kubebuilder deploy docs](https://book-v1.book.kubebuilder.io/beyond_basics/deploying_controller)  
[example how to build operator](https://medium.com/developingnodes/mastering-kubernetes-operators-your-definitive-guide-to-starting-strong-70ff43579eb9)


