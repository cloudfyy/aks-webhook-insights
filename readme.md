# aks-webhook-insights
针对java应用的AKS Mutate Admission Webhook, 

# 要求

1. Kubernetes 1.25.5+
2. Helm 3.11.1+
3. Azure cli 2.45+
4. Docker 23.0.1+ (只有构建镜像才需要)
5. OpenSSL 3.0.2+
6. kubectl v1.26.1+

# 支持的资源类型及操作

本应用支持的资源类型有：

- deployments
- replicasets
- pods

操作类型有：

- Create
- Update

# 工作原理
* scripts/init.sh: 此脚本负责生成webhook应用所需的数字证书。数字证书需要由k8s进行签名然后才能使用。
此脚本还生成部署时所用的helm参数。我们部署时需要把生成的参数合并进values.yaml后再进行部署。
* webhook针对deployment的create事件进行监控。
- (重要)待监控的名字空间需要有 **app-monitoring: enable** 标签;
- 待监控的deploymnet需要有如下注解：

```diff

appinsights.connstr: InstrumentationKey=******;IngestionEndpoint=https://japaneast-1.in.applicationinsights.azure.com/;LiveEndpoint=https://japaneast.livediagnostics.monitor.azure.com/ 
    appinsights.role: (app insights的cloud角色名字)
```

注：Application Insight的cloud role name的含义请参考下方的参考文档。

一个较完整的deployment Yaml例子如下：
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy1
  annotations:
    - appinsights.connstr: InstrumentationKey=******;IngestionEndpoint=https://japaneast-1.in.applicationinsights.azure.com/;LiveEndpoint=https://japaneast.livediagnostics.monitor.azure.com/ 
    appinsights.role: department-service 
```
Pod 的例子yaml如下
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

# 参数说明

helm有如下参数：
|参数名|说明|
|:---|:---|
|namespace|本webhook安装的名字空间。|
|name|webhook名字|
|environment|环境名|
|owner|负责人名字|
|image|webhook镜像地址|
|agents|java application insights镜像地址|
|caBundle|webhook服务器TLS证书公钥。API服务器使用此公钥访问webhook服务器|
|testing|是否为测试环境|
|kVerMajor|Kubernetes主版本号|
|kVerMinor|Kubernetes次版本号|
|kVerRev|Kubernetes修订版本号|
|replicaCount|webhook的pod数目|
|JAVA_TOOL_OPTIONS|此参数会传递到受监控的容器内。我们在此参数中设置java agent。比如：-javaagent:/config/applicationinsights-agent-3.4.10.jar 版本默认为3.4.10。版本信息可以从以下地址获取：https://github.com/microsoft/ApplicationInsights-Java/releases|
|UpdateContainerCmd|是否在container的command命令中添加JAVA_TOOL_OPTIONS参数，默认值 为false|

# 部署方法

1. 克隆本仓库;
2. 首先请检查待监控的命名空间是否有标签app-monitoring=enable。这里以default为例：
```
kubectl label namespace default --list=true

```
如果没有请加上此标签：

```
 kubectl label namespace default app-monitoring=enable
```
如果要删除标签，请使用如下命令：

```
kubectl label namespace default app-monitoring-
```

转到scripts目录，请打开运行init.sh，修改前面部分的参数，参数如何设置请参考参数说明部分。
然后请运行init.sh；

init.sh会生成一个values.yaml文件，请打开查看其内容。
如下为一个例子：
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
我们可以直接使用生成的values.yaml进行部署。

3. 对上一步生成的values.yaml文件修改并保存；
4. 打开deployment/test-deployment.yaml，检查其annotation设置并根据情况进行调整；
5. 转到根目录，运行如下命令安装webhook：

```
 helm install aks-webhook -f ./scripts/values.yaml  ./helm
```
6. 运行如命令安装deployment或者pod：

- Deployment测试场景

```
kubectl apply -f ./deployment/test-deployment.yaml

```
- ReplicaSet测试场景

```
kubectl apply -f ./deployment/test-rs.yaml

```
- Pod测试场景
```
kubectl apply -f ./deployment/test-pod.yaml

```

7. 此时webhook应该已经运行，我们可以运行如下命令查看其结果：

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
8. 如果查看java测试pod，其输出如下：

![测试成功](/img/success.png?raw=true "测试成功")

9. 如果要查看webhook调试信息，请使用如下命令：

得到pod名字
```
kubectl get pods -n kube-system
```
例子结果如下：
```
NAME                                     READY   STATUS    RESTARTS   AGE
ama-logs-gt27g                           2/2     Running   0          14d
ama-logs-rs-7d58796b97-dtfz5             1/1     Running   0          14d
app-monitoring-webhook-858df5c4b-7sllf   1/1     Running   0          18s
...
```
webhook的名字默认为app-monitoring-webhook-*。

我们可以使用如下命令查看其调试信息：

```
 kubectl logs -n kube-system app-monitoring-webhook-858df5c4b-7sllf
```
# 卸载

- Deployment测试场景
```
 helm uninstall aks-webhook
 kubectl delete -f ./deployment/test-deployment.yaml
```
- Replicaset测试场景
```
 helm uninstall aks-webhook
 kubectl delete -f ./deployment/test-rs.yaml
```
- Pod测试场景
```
 helm uninstall aks-webhook
 kubectl delete -f ./deployment/test-pod.yaml
```
# 如何构建镜像

如果需要自己构建镜像，请运行根目录中的build.sh。

# 致谢
本项目参考了lincvic(https://github.com/lincvic)和Microsoft(https://github.com/microsoft)的代码：
- https://github.com/lincvic/aks-webhook-insights
- https://github.com/microsoft/Application-Insights-K8s-Codeless-Attach

Java测试程序基于如下代码：
https://github.com/mag1309/spring-boot-hello-world


# 参考资料

- Dynamic Admission Control(https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
- Configure the Aggregation Layer(https://k8s-docs.netlify.app/en/docs/tasks/access-kubernetes-api/configure-aggregation-layer/#:~:text=Create%20a%20configmap%20in%20the%20kube-system%20namespace%20called,be%20retrieved%20by%20extension%20apiservers%20to%20validate%20requests.)
- Cloud role name(https://learn.microsoft.com/en-us/azure/azure-monitor/app/java-standalone-config#cloud-role-name)

