## tekton-mono-repo-demo

Example project demonstrating a Tekton monorepo setup.

## Setup

Setup Kind cluster with local Docker registry:
```
./create-cluster.sh
```

Install Nginx ingress:
```
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v0.46.0/deploy/static/provider/kind/deploy.yaml
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s
```

Install Tekton Piplines & Triggers:
```
kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.23.0/release.yaml
kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/previous/v0.13.0/interceptors.yaml
kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/previous/v0.13.0/release.yaml
```

Build the interceptor:
```
docker build --tag localhost:5000/interceptor interceptor
docker push localhost:5000/interceptor
```

Apply the pipeline and other manifests:
```
kubectl apply -f manifests/echo-task.yaml\
  -f manifests/interceptor.yaml\
  -f manifests/rbac.yaml\
  -f manifests/service-a-pipeline.yaml\
  -f manifests/service-b-pipeline.yaml
```

Set up the triggers either by applying the example manifests:
```
kubectl apply -f manifests/trigger.yaml
```

Or using the example Helm chart:
```
kubectl apply -f <(helm template test helm/tekton-multi-pipeline-repo --values example-values.yaml)
```

## Hit the webhooks

Webhook for a commit with the sub-folder `service-a` changing:
```
curl localhost/my-repo-trigger --data "{\"repository\":{\"full_name\":\"janakerman/tekton-mono-repo-demo\"},\"before\":\"9f2789c5\",\"after\":\"b30c29fd\"}"
```

Webhook for a commit with the sub-folder `service-b` changing:
```
curl localhost/my-repo-trigger --data "{\"repository\":{\"full_name\":\"janakerman/tekton-mono-repo-demo\"},\"before\":\"b30c29fd\",\"after\":\"c66f7cc1\"}"
```

Webhook for a commit where both sub-folders have changed:
```
curl localhost/my-repo-triggerr --data "{\"repository\":{\"full_name\":\"janakerman/tekton-mono-repo-demo\"},\"before\":\"c66f7cc1\",\"after\":\"3422de7b\"}"
```
