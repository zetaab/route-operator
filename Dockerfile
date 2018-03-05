FROM alpine

USER 1001
COPY bin/linux/route-operator .
ENTRYPOINT ["./route-operator"]
