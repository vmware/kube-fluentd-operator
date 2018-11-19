FROM debian:stretch-slim

RUN apt-get update -y
RUN apt-get install -y gettext

COPY logger.sh /
ENTRYPOINT ["/logger.sh"]
