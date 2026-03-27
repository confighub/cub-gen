FROM eclipse-temurin:21-jre-alpine

WORKDIR /app

COPY target/inventory-api-*.jar app.jar

EXPOSE 8080 8081

ENTRYPOINT ["java", "-jar", "app.jar"]
