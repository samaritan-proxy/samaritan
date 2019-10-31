FROM alpine

RUN mkdir -p /data/run/samaritan
RUN echo $'admin: \n\
  bind: \n\
    ip: 127.0.0.1 \n\
    port: 12345 \n' > /etc/samaritan.yaml
ADD samaritan /usr/local/bin

CMD ["samaritan"]
