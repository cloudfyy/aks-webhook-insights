#!/bin/bash

set -e
title="app-monitoring-webhook"
namespace="kube-system"

[ -z ${title} ] && title=app-monitoring-webhook
[ -z ${namespace} ] && namespace=aks-webhook-ns

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
openssl req -new -key ${tmpdir}/server-key.pem -subj "/CN=${title}.${namespace}.svc /OU=system:nodes /O=system:nodes" -out ${tmpdir}/server.csr -config ${tmpdir}/csr.conf

cat <<EOF >>${tmpdir}/server_cert_ext.cnf
basicConstraints = CA:FALSE
nsCertType = server
nsComment = "OpenSSL Generated Server Certificate for ${title}"
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid,issuer:always
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
EOF

openssl x509 -req -in ${tmpdir}/server.csr  -out ${tmpdir}/server-cert.pem -CAcreateserial -days 3650 -sha256 -extfile ${tmpdir}/server_cert_ext.cnf
#echo ${serverCert} | openssl base64 -d -A -out ${tmpdir}/server-cert.pem

echo "delete the secret if it exists"
kubectl delete secret ${title} -n ${namespace} --ignore-not-found=true
# create the secret with CA cert and server cert/key
echo "create the secret with CA cert and server cert/key"
kubectl create secret generic ${title} \
        --from-file=key.pem=${tmpdir}/server-key.pem \
        --from-file=cert.pem=${tmpdir}/server-cert.pem \
        --dry-run=client -o yaml |
    kubectl -n ${namespace} apply -f -

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

cat <<EOF >> ./values.yaml
app:
  kVerMajor: "${kVerMajor}"
  kVerMinor: "${kVerMinor}"
  kVerRev: "${kVerRev}"
  caBundle: "${CA_BUNDLE}"
EOF
