docker build -t cloudfyy/0.30 .
docker push cloudfyy/0.30

cd tlsgenerator
docker build -t cloudfyy/v02 .
docker push cloudfyy/v02
cd ..
