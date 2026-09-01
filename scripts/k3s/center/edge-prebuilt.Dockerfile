FROM nginx:1.28-alpine

RUN rm -f /etc/nginx/conf.d/default.conf
COPY ide /usr/share/nginx/html
