FROM alpine 

RUN apk --update add ca-certificates

COPY  /dist/drivesync /bin/drivesync
ENTRYPOINT ["/bin/drivesync"]
CMD ["-credentials" "credentials.json" ]