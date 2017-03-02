FROM alpine

COPY build/9volt-linux /

EXPOSE 8080

ENTRYPOINT ["/9volt-linux", "server"]
