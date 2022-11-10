kubectl apply -f .\deployment-svc.yaml
kubectl apply -f .\test-deployment.yaml
kubectl get deploy test-deploy1 -o yaml