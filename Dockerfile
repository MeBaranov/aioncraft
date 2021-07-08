FROM alpine
COPY aioncraft /app/aioncraft
ADD data /app/data
RUN apk add --no-cache libc6-compat
RUN chmod 777 /app/aioncraft
WORKDIR /app
CMD ["./aioncraft"]