FROM BASEIMAGE

# yes this works even if --no-cache gives the impression it would not
RUN apk --no-cache upgrade && \
    apk --no-cache add ca-certificates bash

ARG ARCH
ARG TINI_VERSION

ADD provider /usr/local/bin/crossplane-provider-ionoscloud

EXPOSE 8080
USER 1001
# Add -d for debug mode
ENTRYPOINT ["crossplane-provider-ionoscloud"]
