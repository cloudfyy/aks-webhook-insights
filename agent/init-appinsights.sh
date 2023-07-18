#!/bin/sh 
# This script is used to initialize Application Insights for Java agent
# It will be executed by the agent when the agent is started
# wget https://github.com/microsoft/ApplicationInsights-Java/releases/download/3.4.10/applicationinsights-agent-3.4.10.jar
echo ${CONNECTION_STRING}, ${ROLE_NAME} >debug.log
sed -i "s|ROLE_NAME|${ROLE_NAME}|g" applicationinsights.json
sed -i "s|CONNECTION_STRING|${CONNECTION_STRING}|g" applicationinsights.json