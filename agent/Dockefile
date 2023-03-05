FROM busybox
#ENV APPINSIGHTS_FILE applicationinsights-agent-3.3.1.jar
ENV AGENT_RELEASE 3.4.10
ENV APPINSIGHTS_FILE https://github.com/microsoft/ApplicationInsights-Java/releases/download/${AGENT_RELEASE}/applicationinsights-agent-${AGENT_RELEASE}.jar
ENV APPINSIGHTS_CONFIG applicationinsights.json
ENV INITSHELLFILE init-appinsights.sh
ENV APP_HOME /app 

ADD $APPINSIGHTS_FILE $APP_HOME/
COPY $APPINSIGHTS_CONFIG $APP_HOME/
COPY $INITSHELLFILE $APP_HOME/
WORKDIR $APP_HOME