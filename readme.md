# aks-webhook-insights
AKS Mutate Admission Webhook for Azure Application Insights support

# Things to Know
* Admission Webhook in k8s requires SSL, the **tlsgenerator** generate certificate for this project.
* The generated certificate mounted as a volume in **Mutate Webhook Container**.

# How to deploy
## Option 1: Use pre-build docker image
> Pre-build  **tlsgenerator** docker image : **lincvic/akswebhookcert:v01**
> 
> Pre-build **aks-webhook-insights** docker image : **lincvic/akswebhook:v0.29**
> 
> The pre-defined deploy YAML file : **./deployment/deployment-svc.yaml**

```yaml
kubectl apply -f ./deployment/deployment-svc.yaml
```
> This command uses image **lincvic/akswebhookcert:v01** as init container to generate certificate for Mutate Webhook, 
> deploy image **lincvic/akswebhook:v0.29**, and register the Mutate Webhook to AKS.

## Option 2: Build your own docker image

### Build **aks-webhook-insights** docker image
```shell
docker build -t YOUROWNTAG/WEBHOOKTAG .
docker push YOUROWNTAG/WEBHOOKTAG
```

### Build **tlsgenerator** docker image
```shell
cd tlsgenerator
docker build -t YOUROWNTAG/TLSTAG .
docker push YOUROWNTAG/TLSTAG
```

### Replace the image in **./deployment/deployment-svc.yaml**
```shell
In /spec/template/spec/initContainers/image
Replace lincvic/akswebhookcert:v01 with YOUROWNTAG/TLSTAG

In /spec/template/spec/containers/image
Replace lincvic/akswebhook:v0.29 with YOUROWNTAG/WEBHOOKTAG
```

### Deploy
```yaml
kubectl apply -f ./deployment/deployment-svc.yaml
```

# Usage Example
The Mutate Admission Webhook in  **aks-webhook-insights** only mutate yaml file that contains annotations **appinsights.connstr** and **appinsights.role** in metadata:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy1
  annotations:
    appinsights.connstr: InstrumentationKey=ed9baecd-d00e-46fd-a584-baefb918ca65;IngestionEndpoint=https://japaneast-1.in.applicationinsights.azure.com/;LiveEndpoint=https://japaneast.livediagnostics.monitor.azure.com/
    appinsights.role: department-service
```
### Deploy ./deployment/test-deployment.yaml
```yaml
kubectl apply -f ./deployment/test-deployment.yaml
```

### Check deployed yaml
```yaml
kubectl get deploy test-deploy1 -o yaml
```