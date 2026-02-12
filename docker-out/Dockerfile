FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY fnos-frpc-gui .
VOLUME /app/data
ENV WEB_PORT=7500
ENTRYPOINT ["./fnos-frpc-gui"]
