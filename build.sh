TAG="1.00"

docker build -t cloudfyy/akswebhook:${TAG} .
docker push cloudfyy/akswebhook:${TAG}

cd ../agent
docker build -t cloudfyy/application-insights-java-agent:${TAG} .
docker push cloudfyy/application-insights-java-agent:${TAG}
cd ..
