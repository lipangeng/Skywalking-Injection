BUILD_VERSION   := v1.0.0
BUILD_TIME      := $(shell date "+%F %T")
BUILD_NAME      := skac
SOURCE          := ./
TARGET_DIR      := /usr/local/bin
COMMIT_SHA1     := $(shell git rev-parse HEAD )

all:
	CGO_ENABLED=0
	go build -ldflags '\
    -X "main.BuildVersion=${BUILD_VERSION}" \
    -X "main.BuildTime=${BUILD_TIME}" \
    -X "main.BuildName=${BUILD_NAME}" \
    -X "main.CommitID=${COMMIT_SHA1}" \
    ' \
    -o ${BUILD_NAME} ${SOURCE}

clean:
	rm -rfv ${BUILD_NAME}

install:
	mkdir -p ${TARGET_DIR}
	cp -vf ${BUILD_NAME} ${TARGET_DIR}

.PHONY : all clean install ${BUILD_NAME}