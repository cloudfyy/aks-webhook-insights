#!/bin/bash

set -e

imgver=1.0.29
title="app-monitoring-webhook"
namespace="kube-system"
environment="test"
owner="Microsoft"
testing="false"
replicaCount="1"
image="cloudfyy/akswebhook:${imgver}"
agents="cloudfyy/application-insights-java-agent:${imgver}"
AGENT_RELEASE="3.4.14"
JAVA_TOOL_OPTIONS="-javaagent:/config/applicationinsights-agent-${AGENT_RELEASE}.jar"
UpdateContainerCmd="false"

if [ ! -x "$(command -v kubectl)" ]; then
    echo "kubectl not found"
    exit 1
fi
if [ ! -x "$(command -v openssl)" ]; then
    echo "openssl not found"
    exit 1
fi

csrName=${title}.${namespace}
tmpdir=$(mktemp -d)
echo "creating certs in tmpdir ${tmpdir} "

cat <<EOF >> ${tmpdir}/csr.conf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${title}
DNS.2 = ${title}.${namespace}
DNS.3 = ${title}.${namespace}.svc
DNS.4 = ${namespace}.svc
EOF

openssl genrsa -out ${tmpdir}/server-key.pem 4096
openssl req -new -key ${tmpdir}/server-key.pem -out ${tmpdir}/server.csr -config ${tmpdir}/csr.conf -subj "/CN=system:node:${title}.${namespace}.svc/O=system:nodes"

# clean-up any previously created CSR for our service. Ignore errors if not present.
echo "delete previous csr certs if they exist"
kubectl delete csr ${csrName} 2>/dev/null || true

# create server cert/key CSR and send to k8s API
echo "create server cert/key CSR and send to k8s API"
cat <<EOF | kubectl create -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: ${csrName}
spec:
  groups:
  - system:authenticated
  request: $(cat ${tmpdir}/server.csr | base64 | tr -d '\n')
  signerName: kubernetes.io/kubelet-serving
  expirationSeconds : 315360000 # 10 years
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF

# verify CSR has been created
echo "verify CSR has been created"
while true; do
    kubectl get csr ${csrName}
    if [ "$?" -eq 0 ]; then
        break
    fi
done

# approve and fetch the signed certificate
echo "approve and fetch the signed certificate"
kubectl certificate approve ${csrName}
# verify certificate has been signed

for x in $(seq 10); do
    serverCert=$(kubectl get csr ${csrName} -o jsonpath='{.status.certificate}')
    if [[ ${serverCert} != '' ]]; then
        break
    fi
    sleep 1
done
if [[ ${serverCert} == '' ]]; then
    echo "ERROR: After approving csr ${csrName}, the signed certificate did not appear on the resource. Giving up after 10 attempts." >&2
    exit 1
fi
echo ${serverCert} | openssl base64 -d -A -out ${tmpdir}/server-cert.pem

echo "delete the secret if it exists"
kubectl delete secret ${title} -n ${namespace} --ignore-not-found=true
# create the secret with CA cert and server cert/key
echo "create the secret with CA cert and server cert/key"
kubectl create secret generic ${title} \
        --from-file=key.pem=${tmpdir}/server-key.pem \
        --from-file=cert.pem=${tmpdir}/server-cert.pem \
        --dry-run=client -o yaml |
    kubectl -n ${namespace} apply -f -

# https://k8s-docs.netlify.app/en/docs/tasks/access-kubernetes-api/configure-aggregation-layer/
# The Kubernetes apiserver connects to the extension apiserver over TLS, authenticating itself using a client certificate.
#When started with these options, the Kubernetes apiserver will:

 #   Use them to authenticate to the extension apiserver.
 #   Create a configmap in the kube-system namespace called extension-apiserver-authentication, in which it will place the CA certificate and the allowed CNs. These in turn can be retrieved by extension apiservers to validate requests.

export CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')
#cat ./values._aml | envsubst > ./values.yaml

kVer=`kubectl version --short=true`
kVer="${kVer#*\Server Version: v}"
part1="${kVer%.*}"
kVerMajor="${part1%.*}"
kVerMinor="${part1#*\.}"
kVerRev="${kVer#*\.}"
kVerRev="${kVerRev#*\.}"
echo "found kubernetes server version ${kVer} "

#remove old values.yaml
rm -f ./values.yaml

cat <<EOF >> ./values.yaml
namespace: ${namespace}
app:
  name: "${title}"
  environment: "${environment}"
  owner: "${owner}"
  testing: ${testing}
  image: "${image}"
  agents: "${agents}"
  kVerMajor: "${kVerMajor}"
  kVerMinor: "${kVerMinor}"
  kVerRev: "${kVerRev}"
  caBundle: "${CA_BUNDLE}"
  JAVA_TOOL_OPTIONS: "${JAVA_TOOL_OPTIONS}"
  UpdateContainerCmd: "${UpdateContainerCmd}"
replicaCount: ${replicaCount}
EOF
