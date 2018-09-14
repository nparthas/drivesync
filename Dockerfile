FROM alpine 

RUN apk --update add ca-certificates sudo shadow 

# create new user
ARG user=drivesync
ARG group=drivesync
ARG uid=1000
ARG gid=1000

# add new user to group
RUN addgroup -g ${gid} -S ${group} \
    && adduser -h "/home/${user}" -u ${uid} -g ${gid} -s /bin/bash -S ${user} -G ${group}

# Set up sudo
RUN  usermod -a -G wheel ${user} && echo '%wheel ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers
RUN  usermod -a -G root ${user} 

VOLUME /home/${user}/.drivesync
RUN mkdir -p /home/${user}/.drivesync; \
    touch /home/${user}/.drivesync/drivesync.log; \
    chown ${user}:${group} /home/${user}/.drivesync/drivesync.log

USER ${user} 

COPY  /dist/drivesync /bin/drivesync
ENTRYPOINT ["/bin/drivesync"]
CMD ["-credentials" "credentials.json" ]
