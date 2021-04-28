#!/bin/sh

# 密码
export PASSWORD="lxj"

export COUNTRY="CN"
# 省
export STATE="sc"
# 市
export CITY="cd"
# 公司名称
export ORGANIZATION="lxj"
export ORGANIZATIONAL_UNIT="local"
# 域名
export COMMON_NAME="local-devops-webshell-front.lxj.com"
# 电子邮件地址
export EMAIL="test@qq.com"

export HOST_NAME="$COMMON_NAME"
export IP=`ping $HOST_NAME -c 1 | sed '1{s/[^(]*(//;s/).*//;q}'`

export DIR="cert-$HOST_NAME"

# Workspace

[ -d $DIR ] || mkdir -p $DIR

echo "PASSWORD: $PASSWORD" > $DIR/!nfo.txt
echo "HOST_NAME: $HOST_NAME" >> $DIR/!nfo.txt
echo "HOST_IP: $IP" >> $DIR/!nfo.txt

# Generate CA

openssl genrsa -aes256 -passout "pass:$PASSWORD" -out "$DIR/ca-key.pem" 4096

openssl req -new -x509 -days 3650 -key "$DIR/ca-key.pem" -sha256 -out "$DIR/ca.pem" -passin "pass:$PASSWORD" -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORGANIZATION/OU=$ORGANIZATIONAL_UNIT/CN=$COMMON_NAME/emailAddress=$EMAIL"

# Generate Server Certs

openssl genrsa -out "$DIR/server-key.pem" 4096

openssl req -subj "/CN=$HOST_NAME" -sha256 -new -key "$DIR/server-key.pem" -out $DIR/server.csr

echo "subjectAltName = DNS:$HOST_NAME,IP:$IP,IP:127.0.0.1" > $DIR/server.cnf
echo "extendedKeyUsage = serverAuth" >> $DIR/server.cnf

openssl x509 -req -days 3650 -sha256 -in $DIR/server.csr -passin "pass:$PASSWORD" -CA "$DIR/ca.pem" -CAkey "$DIR/ca-key.pem" -CAcreateserial -out "$DIR/server-cert.pem" -extfile $DIR/server.cnf

# Generate Client Certs

openssl genrsa -out "$DIR/client-key.pem" 4096

openssl req -subj '/CN=client' -new -key "$DIR/client-key.pem" -out $DIR/client.csr

echo "extendedKeyUsage = clientAuth" > $DIR/client.cnf

openssl x509 -req -days 3650 -sha256 -in $DIR/client.csr -passin "pass:$PASSWORD" -CA "$DIR/ca.pem" -CAkey "$DIR/ca-key.pem" -CAcreateserial -out "$DIR/client-cert.pem" -extfile $DIR/client.cnf

# Modify Certs Permission

chmod 0400 $DIR/*-key.pem
chmod 0444 $DIR/ca.pem $DIR/*-cert.pem

# Remove Temporary Files

rm -f $DIR/*.csr $DIR/*.cnf

# Install To Docker Daemon

mkdir -p /etc/docker/certs.d
cp $DIR/ca.pem /etc/docker/certs.d/
cp $DIR/server-*.pem /etc/docker/certs.d/

#cat <<EOF >/etc/docker/daemon.json
#{
#    "tlsverify": true,
#    "tlscacert": "/etc/docker/certs.d/ca.pem",
#    "tlscert": "/etc/docker/certs.d/server-cert.pem",
#    "tlskey": "/etc/docker/certs.d/server-key.pem",
#    "hosts": ["unix:///var/run/docker.sock", "tcp://127.0.0.1:2375"]
#}
#EOF

# Modify Systemd Service

#if [ -f /lib/systemd/system/docker.service ]; then
#    sed -i 's#\["tcp:#\["fd://", "tcp:#' /etc/docker/daemon.json
#    sed -i 's# -H fd://##' /lib/systemd/system/docker.service
#    systemctl daemon-reload && systemctl restart docker
#fi
