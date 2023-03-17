TAG="1.0.7"

cd ./agent
docker build -t cloudfyy/application-insights-java-agent:${TAG} .
docker push cloudfyy/application-insights-java-agent:${TAG}
cd ..
docker build -t cloudfyy/akswebhook:${TAG} .
docker push cloudfyy/akswebhook:${TAG}

cd ../javatest
docker build -t cloudfyy/akswebhookjavademo:2.0 -f multi-stage.Dockerfile .
docker push cloudfyy/akswebhookjavademo:2.0



