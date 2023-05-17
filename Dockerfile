FROM golang:1.20

RUN mkdir -p /app \
    && chown -R nobody:nogroup /app
WORKDIR /app

RUN apt-get update && apt-get install -y curl make gcc g++
ENV GOPROXY https://goproxy.cn,direct

COPY . src 
RUN cd src \
    && go build \
    && cd .. \
    && mv src/utxo-indexer /app/utxo-indexer \
    && mkdir -p /app/conf \
    && mv src/config.yaml /app/conf/config.yaml \
    && rm -rf src 
EXPOSE 3000
VOLUME /app/conf

CMD ["/app/utxo-indexer", "-conf", "/app/conf/config.yaml"]