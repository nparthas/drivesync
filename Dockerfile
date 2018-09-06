FROM alpine 

COPY  /dist/drivesync /bin/drivesync
CMD ["/bin/drivesync"]