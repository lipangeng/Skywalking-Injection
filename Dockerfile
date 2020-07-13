FROM dhub.msorg.cn/library/alpine

# Add Tini
ENV TINI_VERSION v0.19.0
RUN set -ex; \
	\
	apk add --no-cache curl ;\
	\
    curl -L -o /usr/local/bin/tini https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini


FROM dhub.msorg.cn/library/alpine
MAINTAINER 李盼庚<lipg@outlook.com>

COPY --from=0 /usr/local/bin/tini /bin/tini

COPY ./skac /usr/bin/

RUN set -eux ;\
	\
	ls -la /usr/bin/ \
	\
	; chmod +x /usr/bin/skac
#    \
#    ; /usr/bin/skac --version

ENTRYPOINT ["/bin/tini", "--"]

CMD /usr/bin/skac $SKAC_OPTIONS