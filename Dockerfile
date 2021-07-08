FROM alpine
ARG token
ARG gcs
COPY aioncraft /app/aioncraft
ADD data /app
RUN apk add --no-cache libc6-compat
RUN chmod 777 /app/aioncraft
ENV BOT_TOKEN=$token
ENV GCS_BUCKET=${gcs}
CMD ["/app/aioncraft"]