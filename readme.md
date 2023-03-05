# aks-webhook-insights
针对java应用的AKS Mutate Admission Webhook, 

# 要求

1. Kubernetes 1.25.5+
2. Helm 3.11.1+
3. Azure cli 2.45+
4. Docker 23.0.1+ (只有构建镜像才需要)
5. OpenSSL 3.0.2+
6. kubectl v1.26.1+

# 工作原理
* scripts/init.sh: 此脚本负责生成webhook应用所需的数字证书。数字证书需要由k8s进行签名然后才能使用。
此脚本还生成部署时所用的helm参数。我们部署时需要把生成的参数合并进values.yaml后再进行部署。
* webhook针对deployment的create事件进行监控。
- (重要)待监控的名字空间需要有 app-monitoring: enable 标签;
- 待监控的deploymnet需要有如下注解：

```diff

appinsights.connstr: InstrumentationKey=******;IngestionEndpoint=https://japaneast-1.in.applicationinsights.azure.com/;LiveEndpoint=https://japaneast.livediagnostics.monitor.azure.com/ 
+    appinsights.role: department-service 
```
一个较完整的deployment Yaml如下：
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deploy1
  annotations:
    - appinsights.connstr: InstrumentationKey=******;IngestionEndpoint=https://japaneast-1.in.applicationinsights.azure.com/;LiveEndpoint=https://japaneast.livediagnostics.monitor.azure.com/ 
    appinsights.role: department-service 
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
|javaagentoptions|Application insight java包参数配置。比如：-javaagent:applicationinsights-agent-3.4.10.jar|
|javastartpackage|应用Java包启动配置。比如：-jar department-service-1.2-SNAPSHOT.jar|

# 部署方法

1. 克隆本仓库;
2. 转到scripts目录，请打开运行init.sh，修改前面部分的参数，参数如何设置请参考参数说明部分。然后请运行init.sh；
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
  caBundle:  "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUU2VENDQXRHZ0F3SUJBZ0lSQVBibDdpZGp5NzZGNWltaEMwTW5BaWd3RFFZSktvWklodmNOQVFFTEJRQXcKRFRFTE1Ba0dBMVVFQXhNQ1kyRXdJQmNOTWpJeE1ESTBNRGcwTmpRM1doZ1BNakExTWpFd01qUXdPRFUyTkRkYQpNQTB4Q3pBSkJnTlZCQU1UQW1OaE1JSUNJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBZzhBTUlJQ0NnS0NBZ0VBCnJ2QVZoYk9SV3c0QjFMYVI0emhQdDh0YlVqTU5XQ0NzYWJUL2lhOUJMVzlYQmU0dWpreWd4Yjg2UTVGeEY5TjYKejhmdHlTRlQzY2tsVjlseVNWY01Pc21wZEsvMWc1MENmQ0tyYk5CVTQrLzJVSjllOEZzSm0xN3lVcG1VTnA4cgoreFJteG9WRC8yOHN0OUJ6U09jTUd3Rmt0Qk94Wm1pTUs5NCszREx1Q1dPdkdUOXpuQ1lDS1VZUzZQdVliclh4CnpqTy8vWnBqalJheXQ4c0Q4eVQ5K3ZZQUxGVGRJejJMMzhNMkhYKzlXOUNOVWVuM3FYSDFBRDlpclV1Ump2ZXgKUzYzdW1rQlk1ZnVaQU5hemZuODJGLzllOE1vRkYyMWszN3ZaL0xiZm1SQUNMYmNOY0pFcHBtWkpRM2NKSDVQagp5eklhN1BqcE8valdoNXRoWll5Nkh0aTVzelZ0L1RqaDMvak9wT2lxVEVkQmFMVi9WSG5GSXBWM1pHMWRVVThRCjcyd2pWUExxbSsyZFJmVTM1ZWVJWkpTWDZSSmRQKzM4UVdEbzJES3ZCcXpueGNyNC9icllXSCtiMG9QZU5mNWoKUnJjZTRpRDY1YkJqZUt3eEx4YzUrVjRTTyt3OFl5TGFqakI4TkJwRUdqd3dmS0lKRkJIQUFIdjdILzJjRkJGeQpNc2RUVE9sZW9IanhsQ25uTldVUjE0elVNMmc1RldZVXIwM3FUaHY5a1B1dXV4OWRsM1BLTEFpOVZhb3BCZXNvCmhmOUhCdTdKVzVnNW93WUdIeFRXWFBwWS9zOXJRUlhISWh2QVhQV3hpQzVkWkpVOTY1cTIzUlBWWkgrZTNpb2wKclBabGRucVJXME5VZCtSTUQ3Wi9pVjJXTnV4ejRMREFKT254aUdmdFdma0NBd0VBQWFOQ01FQXdEZ1lEVlIwUApBUUgvQkFRREFnS2tNQThHQTFVZEV3RUIvd1FGTUFNQkFmOHdIUVlEVlIwT0JCWUVGRis3eHZONXZUcFg5ZGIwClJaWU16TFlqdmt1TU1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQ0FRQUQwMUdlanZLQklSRmgrb3Z3VEVLVmh2NG0KZmlhY2I3RkFYaWx1MW9BUE5aMWxEWGQrd1dzeHVoQlJUckdRaDdCVFBsdG15SkpyVEIzNVp2bnI1bTRLei9DNApwYlQxZFRDSGV1aG8wZmo1cVkyNG1vOENselNNRE1VcDdyTTdGT0gwOEJWYjU4RGl1ZVFiS1FuQ3RSbG1CR2twCjAreE9GT2V0Y3kySXE2cW1abVJ6dDN3TzBNRTdIZHladU1LYjZhYUFkZ241SnhHN0JnNG1ab050MkxwMjRWWEcKU1JZYVVYU3JiUW03NmtyZFFDR01rN3htWFljdktXTkg3aTJURGtXTG1VK3hMZHRBKytZV1dGUjNNdWIxQWJobAp3M2VHUU5NYXVRaDMxVzcxZUYxdThnS1RvRm5FRVhjVkZIZVBqc0NpMHUxU3MzL1JFdkVDQVltdUlVQldZYjAvCjJDWkpNNUlIOUdQbmtKNFNqK0JKQ0JRckpnZXlORC9TZGZteGpoMUFvSnk5NTZnS2pXTzZQUEY0NFRiWHJqV0sKUE9kRmZ2aWdGdXlnVkFBY2xjSGNXc1VXNWRQQW1weU9PRHFJV055N25vNXFJcU1Fc1E4a29vdEczZDQzNVJsSgp4dE5RNGdpd0hPOVlTMlBOWXNyaTAvWTU1YVNncmxRYjZqTXZuRVZENFQxd2VNRm8wRUZnZ0dzVG9jRnNLaVlZCkwvUGdudGlqdUUwTlFpR1pvcTBMOW43d0ZzOUdFbHlvRXNKbnAybitobTZNRC9TMmptUEFnb04xcWVsSURndE0KdlU3TTB1NWFYeUczbVZnUytnOHU4RXcyNmQzZTNwV2dCOW1uUlE3NXR1MFZWc2tQNUZudS9LVkdwYVlVUG1uQgowOW10K2o4ZjVOL0c2dXVlMGc9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
    
  testing: false
  kVerMajor: "1"
  kVerMinor: "25"
  kVerRev: "5"
  
replicaCount: 1
```
我们可以直接使用生成的values.yaml进行部署。

3. 我们可以直接使用上一步生成的values.yaml文件；
4. 打开deployment/test-deployment.yaml，检查其annotation设置并根据情况进行调整；
5. 运行如下命令安装webhook：

```
 helm install aks-webhook -f ./scripts/values.yaml  ./helm
```
6. 运行如命令安装deployment：

```
kubectl apply -f ./deployment/test-deployment.yaml

```
7. 此时webhook应该已经运行，我们可以运行如下命令查看其结果：

```
kubectl get deploy test-deploy1 -o yaml
```
8. 如果要查看webhook调试信息，请使用如下命令：

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

# 如何构建镜像

如果需要自己构建镜像，请运行根目录中的build.sh。

# 致谢
本项目参考了lincvic(https://github.com/lincvic)和Microsoft(https://github.com/microsoft)的代码：
- https://github.com/lincvic/aks-webhook-insights
- https://github.com/microsoft/Application-Insights-K8s-Codeless-Attach

