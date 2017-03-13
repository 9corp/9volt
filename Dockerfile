FROM alpine

# Because we need /etc/ssl/* for https:// checks
RUN apk --no-cache add ca-certificates && update-ca-certificates

COPY build/9volt-linux /

EXPOSE 8080

ENTRYPOINT ["/9volt-linux", "server"]
