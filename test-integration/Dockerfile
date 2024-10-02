# Use the docker:dind image as the base image
FROM docker:dind

# Install curl and bash
RUN apk update && apk add --no-cache curl bash

# Download and execute the Bash script from the given URL
RUN curl -sSL https://get.bacalhau.org/install.sh | bash

ENTRYPOINT ["dockerd-entrypoint.sh"]
