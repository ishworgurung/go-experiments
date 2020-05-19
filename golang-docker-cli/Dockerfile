FROM debian:buster as stage1

RUN echo hello > /tmp/hello


FROM alpine:latest as stage2
COPY --from=stage1 /tmp/hello /hello
RUN cat /hello

