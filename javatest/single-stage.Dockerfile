# initialize build and set base image
FROM maven:3.6.3-adoptopenjdk-11 as base

# set working directory
WORKDIR /app

# copy project files
COPY . .

# compile the source code and package it in a jar file
RUN mvn clean package -Dmaven.test.skip=true
ENTRYPOINT  ["/bin/sh", "-c", "java", "-jar demo.jar"]