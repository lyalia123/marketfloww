FROM alpine:3.18

RUN apk add --no-cache docker-cli bash

COPY cmd/docker/tar_files /app/tar_files
COPY cmd/docker/start_exchanges.sh /app/

WORKDIR /app

RUN chmod +x /app/start_exchanges.sh

CMD ["/bin/bash", "/app/start_exchanges.sh"]