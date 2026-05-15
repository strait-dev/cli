# STRAIT_TEMPLATE_PROJECT_NAME

Strait worker for Kubernetes. Maintains a long-lived gRPC stream to the
orchestrator and pulls tasks for the queues you subscribe to.

## Local

```bash
npm install
export STRAIT_API_KEY=<your-key>
export STRAIT_QUEUES=default
npm run dev
```

## Build + push image

```bash
docker build -t STRAIT_TEMPLATE_PROJECT_NAME:latest .
docker tag STRAIT_TEMPLATE_PROJECT_NAME:latest <registry>/STRAIT_TEMPLATE_PROJECT_NAME:latest
docker push <registry>/STRAIT_TEMPLATE_PROJECT_NAME:latest
```

## Deploy to Kubernetes

```bash
kubectl create secret generic strait-credentials --from-literal=api-key=<your-key>
kubectl apply -f k8s/deployment.yaml
```

## Register jobs

```bash
strait deploy push   # upserts src/jobs.ts to the orchestrator
strait worker status # confirms your worker pod has connected
```
