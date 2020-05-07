  # -p 8000:80 \
  # -e VIRTUAL_HOST=localhost:8000 \

docker run -d -i \
  --name pihole \
  -p 53:53/tcp \
  -p 53:53/udp \
  -p 80:80 \
  -e TZ="Asia/Tbilisi" \
  -e WEBPASSWORD="1234" \
  -v $(pwd)/volume/etc/:/etc/pihole/ \
  -v $(pwd)/volume/dnsmasq.d/:/etc/dnsmasq.d/ \
  --dns=0.0.0.0 --dns=1.1.1.1 \
  pihole/pihole:latest
