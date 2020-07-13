FROM dhub.msorg.cn/library/golang
MAINTAINER 李盼庚<lipg@outlook.com>

RUN set -eux \
	; ls -la \
	; make \
	; make install


FROM dhub.msorg.cn/library/alpine
MAINTAINER 李盼庚<lipg@outlook.com>

COPY --from=0 /usr/local/bin/skac /usr/local/bin/skac

# Add Tini
ENV TINI_VERSION v0.19.0
RUN set -ex; \
    curl -L -o /usr/local/bin/tini https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini \
    ; chmod +x /usr/local/bin/tini \
    ; ln -s /usr/local/bin/tini /bin/tini

ENTRYPOINT ["/bin/tini", "--"]

CMD skac $SKAC_OPTIONS