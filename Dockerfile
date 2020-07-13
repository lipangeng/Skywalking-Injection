FROM dhub.msorg.cn/library/busybox

# Add Tini
ENV TINI_VERSION v0.19.0
RUN set -ex; \
	\
    curl -L -o /usr/local/bin/tini https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini


FROM dhub.msorg.cn/library/alpine
MAINTAINER 李盼庚<lipg@outlook.com>

COPY --from0 /usr/local/bin/tini /bin/tini

COPY skac /usr/bin/skac

RUN set -eux ;\
    \
    skac --version

ENTRYPOINT ["/bin/tini", "--"]

CMD /usr/bin/skac $SKAC_OPTIONS