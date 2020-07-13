FROM dhub.msorg.cn/library/alpine
MAINTAINER 李盼庚<lipg@outlook.com>

COPY skac /usr/bin/skac

# Add Tini
ENV TINI_VERSION v0.19.0
RUN set -ex; \
	\
    curl -L -o /usr/local/bin/tini https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini \
    ; chmod +x /usr/local/bin/tini \
    ; ln -s /usr/local/bin/tini /bin/tini \
    \
    ; skac --version

ENTRYPOINT ["/bin/tini", "--"]

CMD /usr/bin/skac $SKAC_OPTIONS