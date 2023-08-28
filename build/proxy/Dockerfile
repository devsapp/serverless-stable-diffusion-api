FROM centos:latest
COPY build/proxy/proxyServer /proxy/proxy
COPY proxy.yaml /proxy/proxy.yaml
COPY build/proxy/entrypoint.sh /docker/entrypoint.sh
EXPOSE 80
ENTRYPOINT ["/docker/entrypoint.sh"]