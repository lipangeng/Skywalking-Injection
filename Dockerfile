FROM golang:alpine

ADD ./ /app/

WORKDIR /app

RUN set -eux; \
	\
	apk add --no-cache make git \
	; ls -la \
	; make \
	; ./skac --version


FROM alpine
MAINTAINER 李盼庚<lipg@outlook.com>

COPY --from=0 /app/skac /usr/bin/

RUN set -eux ;\
	\
	apk add --no-cache tini \
	\
	; chmod +x /usr/bin/skac \
    \
    ; /usr/bin/skac --version

ENTRYPOINT ["/sbin/tini", "--"]

CMD /usr/bin/skac $SKAC_OPTIONS