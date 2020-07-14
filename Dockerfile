FROM dhub.msorg.cn/library/alpine
MAINTAINER 李盼庚<lipg@outlook.com>

COPY skac /usr/bin/

RUN set -eux ;\
	\
	sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
	; apk add --no-cache tini \
	\
	; chmod +x /usr/bin/skac \
    \
    ; /usr/bin/skac --version

ENTRYPOINT ["/sbin/tini", "--"]

CMD /usr/bin/skac $SKAC_OPTIONS