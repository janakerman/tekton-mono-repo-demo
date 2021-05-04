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

Apply the manifests:
```
kubectl apply -f manifests/
```
