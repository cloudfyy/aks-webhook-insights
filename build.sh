docker build -t cloudfyy/akswebhook:0.30 .
docker push cloudfyy/akswebhook:0.30

cd tlsgenerator
docker build -t cloudfyy/akswebhookcert:v02 .
docker push cloudfyy/akswebhookcert:v02
cd ..
