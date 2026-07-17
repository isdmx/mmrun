FROM scratch
COPY mmrun /mmrun
COPY --from=alpine:latest /etc/ssl/cert.pem /etc/ssl/cert.pem
ENTRYPOINT ["/mmrun"]
