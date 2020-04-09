#
# BUILD STEP
#
# FROM riftbit/ffalpine:latest-go as build-stage
#
# ARG GFM_REPO=3d0c/gmf
#
# RUN go get github.com/${GFM_REPO}
#
# WORKDIR $GOPATH/src/github.com/${GFM_REPO}/examples
#
# RUN mkdir -p /examples && cp ./bbb.mp4 /examples/ && \
#     go build -o /examples/stress stress.go
#
# #
# # RUNTIME STEP
# #
# FROM riftbit/ffalpine:latest
#
# COPY --from=build-stage /examples /examples
#
# RUN ls -la /examples
#
# WORKDIR /examples
# ENTRYPOINT ["./stress"]

FROM riftbit/ffalpine:latest-go


WORKDIR /extractor

COPY go.mod .
COPY go.sum .

RUN go mod download


ADD cmd /extractor/cmd
ADD pkg /extractor/pkg


RUN cd cmd && go build -o extractor .
RUN adduser -S -D -H -h /extractor appuser
USER appuser
ENTRYPOINT ["cmd/extractor"]