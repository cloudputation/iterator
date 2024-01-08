FROM alpine:3


ARG NAME=iterator
ARG SERVICE_USERNAME=iterator
ARG PRODUCT_VERSION=0.0.2

ENV NAME=$NAME
ENV VERSION=$PRODUCT_VERSION
ENV ROOTDIR="/iterator"
ENV ITERATOR_CONFIG_FILE_PATH="/etc/iterator/config.hcl"
ENV ITERATOR_LOG_DIRECTORY="/var/log/iterator"
ENV ITERATOR_DATA_DIRECTORY="/var/lib/iterator"
ENV TERRAFORM_PATH="/usr/local/bin/terraform"
ENV TERRAGRUNT_PATH="/usr/local/bin/terragrunt"

WORKDIR ${ROOTDIR}

# Install runtime dependencies
RUN apk add --no-cache dumb-init git

# Create service directories
RUN mkdir -p /etc/iterator \
    && mkdir -p ${ITERATOR_LOG_DIRECTORY} \
    && mkdir -p ${ITERATOR_DATA_DIRECTORY}

# Set service user
RUN addgroup -g 991 ${SERVICE_USERNAME} \
    && adduser -D -u 991 -G ${SERVICE_USERNAME} ${SERVICE_USERNAME}

# Copy artifacts from builder
COPY ./API_VERSION ./API_VERSION
COPY ./artifacts/terraform ${TERRAFORM_PATH}
COPY ./artifacts/terragrunt ${TERRAGRUNT_PATH}
COPY ./build/iterator /bin/iterator
COPY ./.release/defaults/test.config.hcl /etc/iterator/config.hcl
COPY .release/docker/docker-entrypoint.sh /bin/docker-entrypoint.sh

# Set permissions
RUN chown -R ${SERVICE_USERNAME}:${SERVICE_USERNAME} ${ROOTDIR} \
    && chown -R ${SERVICE_USERNAME}:${SERVICE_USERNAME} ${ITERATOR_LOG_DIRECTORY} \
    && chown -R ${SERVICE_USERNAME}:${SERVICE_USERNAME} ${ITERATOR_DATA_DIRECTORY} \
    && chmod +x /bin/docker-entrypoint.sh \
    && chmod +x ${TERRAFORM_PATH} \
    && chmod +x ${TERRAGRUNT_PATH}

# Expose port 9595
EXPOSE 9595

# Set user
USER ${SERVICE_USERNAME}

# Entrypoint to run the executable
ENTRYPOINT ["/bin/docker-entrypoint.sh"]
CMD ["/bin/iterator"]
