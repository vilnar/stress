FROM php:7.2-cli
RUN [ "apt-get", "update" ]
RUN [ "apt-get", "-y", "install", "netcat" ]
RUN [ "docker-php-ext-install", "sysvsem" ]
RUN [ "docker-php-ext-install", "sockets" ]
RUN [ "docker-php-ext-install", "pcntl" ]
RUN [ "docker-php-ext-install", "mysqli" ]
WORKDIR /app
COPY . /app
#ENTRYPOINT [ "php", "./test.php" ]
CMD [ "php", "./test.php" ]
