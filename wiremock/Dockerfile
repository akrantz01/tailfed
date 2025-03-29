ARG WIREMOCK_VERSION=3.12.1
FROM wiremock/wiremock:${WIREMOCK_VERSION}

ARG WIREMOCK_JWT_VERSION=0.2.0
RUN mkdir -p /var/wiremock/extensions && \
    curl https://repo1.maven.org/maven2/org/wiremock/extensions/wiremock-jwt-extension-standalone/${WIREMOCK_JWT_VERSION}/wiremock-jwt-extension-standalone-${WIREMOCK_JWT_VERSION}.jar \
        -o /var/wiremock/extensions/wiremock-jwt-extension-standalone.jar

ENTRYPOINT ["/docker-entrypoint.sh", "--global-response-templating"]
