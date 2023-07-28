# aks-webhook-insights [中文版](./readme_cn.md)
A codeless AKS Mutate Admission Webhook for Java Application 

# Prerequisuites

1. Kubernetes 1.25.5+
2. Helm 3.11.1+
3. Azure cli 2.45+
4. Docker 23.0.1+ (requried when build image by yourself)
5. OpenSSL 3.0.2+
6. kubectl v1.26.1+

# Supported Resource Types and Operations

This webkhook supports the following Kubernetes resources：

- deployments
- replicasets
- pods

supported Operations：

- Create
- Update

# Description
* scripts/init.sh: When you deploy this webhook, you must init.sh at first. This script checks current kubernetes infomation, creates a cretificate which is used by webhook and then created a helm values.yaml file, which is used by helm installation.

* webhook monitors one or more namespaces:
- (important)the neamesapce to be monitored must has  **app-monitoring: enable** label;
- The monitored deployments, replica sets and containers must has the following anonations：

```diff

appinsights.connstr: InstrumentationKey=******;IngestionEndpoint=https://japaneast-1.in.applicationinsights.azure.com/;LiveEndpoint=https://japaneast.livediagnostics.monitor.azure.com/ 
    appinsights.role: (app insights的cloud角色名字)
```

For more infomation of conection string, cloud role name of application insights, please refer to the documents in the acknowledgment sections.

This is sample deployment yaml：
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy1
  annotations:
    - appinsights.connstr: InstrumentationKey=******;IngestionEndpoint=https://japaneast-1.in.applicationinsights.azure.com/;LiveEndpoint=https://japaneast.livediagnostics.monitor.azure.com/ 
    appinsights.role: department-service 
```
This is sample pod
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  annotations:
    appinsights.connstr: InstrumentationKey=********;IngestionEndpoint=https://eastasia-0.in.applicationinsights.azure.com/;LiveEndpoint=https://eastasia.livediagnostics.monitor.azure.com/
    appinsights.role: department-service
spec:
  containers:
  - name: javademo-mutate
    image: cloudfyy/akswebhookjavademo:2.2
    imagePullPolicy: Always
    ports:
    - containerPort: 8080
    command: ["/bin/sh", "-c", "java -jar /opt/demo/demo.jar"]

```

# Helm Parameter


|Name|Description|
|:---|:---|
|namespace|The namespace where this webhook will be installed|
|name|webhook name|
|environment|You can gave a name to this environment. |
|owner|who own this environment|
|image|webhook image url. if you build your own image, you must change this value.|
|agents|java application insights image url. if you build your own image, you must change this value.|
|caBundle|webhook server TLS certificate public key. It is creaed by init.sh|
|testing|Is this environment a test environment. For test environment, we only create 1 replica|
|kVerMajor|Kubernetes major version|
|kVerMinor|Kubernetes minor version|
|kVerRev|Kubernetes revision version|
|replicaCount|the pod number of this webhook|
|JAVA_TOOL_OPTIONS|Java container JAVA_TOOL_OPTIONS environment value. for example：-javaagent:/config/applicationinsights-agent-3.4.10.jar 版本默认为3.4.10。To get the latest jave agent version info, please visit：https://github.com/microsoft/ApplicationInsights-Java/releases|
|UpdateContainerCmd|Whether add JAVA_TOOL_OPTIONS parameter to container command line，the default value is false|

# Deploy

1. clone this repository;
2. Check where the monitored namesapce labed with app-monitoring=enable or not:

the command below is and example of the default namespace
```
kubectl label namespace default --list=true

```
If not, please add the label：

```
 kubectl label namespace default app-monitoring=enable
```
if you want to delete the label, please run this command：

```
kubectl label namespace default app-monitoring-
```

Switch to scripts folder, then run init.sh. Then you can open values.yaml to modifiy it.


This is a sample values.yaml：
```yaml
namespace: "kube-system"

app:
  name: "app-monitoring-webhook"
  environment: "test"
  owner : "Microsoft"
  image: "cloudfyy/akswebhook:1.00"
  agents: "nikawang.azurecr.io/spring/app-insights-agent:v1"
  caBundle:  "****"
    
  testing: false
  kVerMajor: "1"
  kVerMinor: "25"
  kVerRev: "5"
  
replicaCount: 1
```


3. Save values.yaml；
4. Switch back to parent folder. then please open deployment/test-deployment.yamland check the annotation settings. You may want to change it accroding to your application insights settings；
5. run the follwoing comamnd to install webhook：

```
 helm install aks-webhook -f ./scripts/values.yaml  ./helm
```
6. Run the following command to install test deployment/replicaset/pod：

- Deployment Test

```
kubectl apply -f ./deployment/test-deployment.yaml

```
- ReplicaSet Test

```
kubectl apply -f ./deployment/test-rs.yaml

```
- Pod Test
```
kubectl apply -f ./deployment/test-pod.yaml

```

7. We can run the following command to verfiy the results：

- Deployment
```
kubectl get deploy java-test-deploy -o yaml
```

- Replicatset
```
kubectl get rs test-rs -o yaml
```

- Pod

```
kubectl get pod test-pod -o yaml
```
8. This is a successful pod test results：

![测试成功](/img/success.png?raw=true "测试成功")

9. If you want to view webhook debug trace, please run the following commands：

get webhoos pod name
```
kubectl get pods -n kube-system
```
sample output：
```
NAME                                     READY   STATUS    RESTARTS   AGE
ama-logs-gt27g                           2/2     Running   0          14d
ama-logs-rs-7d58796b97-dtfz5             1/1     Running   0          14d
app-monitoring-webhook-858df5c4b-7sllf   1/1     Running   0          18s
...
```
In this output, the webhook pod name is app-monitoring-webhook-*。

The we use this command to view its log：

```
 kubectl logs -n kube-system app-monitoring-webhook-858df5c4b-7sllf
```
# Unisntall

- Deployment Test
```
 helm uninstall aks-webhook
 kubectl delete -f ./deployment/test-deployment.yaml
```
- Replicaset est
```
 helm uninstall aks-webhook
 kubectl delete -f ./deployment/test-rs.yaml
```
- Pod Test
```
 helm uninstall aks-webhook
 kubectl delete -f ./deployment/test-pod.yaml
```
# Build

If you want to build images by yourself, please update and run the build.sh in the root folder.


# Acknowledge
I use the code of lincvic(https://github.com/lincvic) and Microsoft(https://github.com/microsoft)：
- https://github.com/lincvic/aks-webhook-insights
- https://github.com/microsoft/Application-Insights-K8s-Codeless-Attach

Java test application code is modified from this repository：
https://github.com/mag1309/spring-boot-hello-world


# Reference

- Dynamic Admission Control(https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
- Configure the Aggregation Layer(https://k8s-docs.netlify.app/en/docs/tasks/access-kubernetes-api/configure-aggregation-layer/#:~:text=Create%20a%20configmap%20in%20the%20kube-system%20namespace%20called,be%20retrieved%20by%20extension%20apiservers%20to%20validate%20requests.)
- Cloud role name(https://learn.microsoft.com/en-us/azure/azure-monitor/app/java-standalone-config#cloud-role-name)

