TAG="0.40"

docker build -t cloudfyy/akswebhook:${TAG} .
docker push cloudfyy/akswebhook:${TAG}

#cd tlsgenerator
#docker build -t cloudfyy/akswebhookcert:v02 .
#docker push cloudfyy/akswebhookcert:v02
#cd ..
