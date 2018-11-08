FROM php:7.2-cli
WORKDIR /app
COPY . /app
ENTRYPOINT [ "php", "./test.php" ]
CMD [ "-h" ]
