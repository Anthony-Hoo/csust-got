# __IMPORTENT__ 
# requires docker 17.3 and above

# build
FROM golang:alpine AS buildenv

RUN apk add make git tzdata

ARG BRANCH
ARG TAG
ARG RELEASE

ENV BRANCH=$BRANCH
ENV TAG=$TAG

WORKDIR /go/src/app
COPY . .
RUN make deploy

# deploy image
FROM alpine

RUN apk add tzdata

WORKDIR /app
COPY --from=buildenv /go/src/app/got .
COPY --from=buildenv /go/src/app/config.yaml .

CMD ["./got"]
