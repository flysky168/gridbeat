FROM busybox:stable
ARG TARGETARCH

HEALTHCHECK --interval=30s --timeout=10s --retries=3 CMD curl --fail http://localhost:8080/health || exit 1

RUN mkdir -p /var/log/gridbeat /var/lib/gridbeat/data /var/lib/gridbeat/extra /var/lib/gridbeat/plugins

COPY linux/${TARGETARCH}/gridbeat /gridbeat
#COPY plugins/*.so /usr/lib/gridbeat/plugins/

VOLUME /var/log/gridbeat /var/lib/gridbeat/data /var/lib/gridbeat/extra

EXPOSE 8080

RUN  chmod +x /gridbeat
CMD ["/gridbeat","server"]
