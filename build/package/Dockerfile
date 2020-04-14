# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
USER nonroot:nonroot

# Add the binary
COPY manager /manager

ENTRYPOINT ["/manager"]
