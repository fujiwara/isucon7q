#!/bin/sh
# for Ubuntu

gpg --keyserver  hkp://keys.gnupg.net --recv-keys 1C4CBDCDCD2EFD2A
gpg -a --export CD2EFD2A | sudo apt-key add -
echo "deb http://repo.percona.com/apt `lsb_release -cs` main" >> /etc/apt/sources.list.d/percona.list
echo "deb-src http://repo.percona.com/apt `lsb_release -cs` main" >> /etc/apt/sources.list.d/percona.list

apt-get update
apt-get -y --allow-unauthenticated install ruby ruby-dev gcc make build-essential vim ack-grep silversearcher-ag percona-toolkit tig build-essential make libpcre3 libpcre3-dev libgeoip-dev
wget -O /tmp/chef_12.21.14-1_amd64.deb "https://packages.chef.io/files/stable/chef/12.21.14/ubuntu/16.04/chef_12.21.14-1_amd64.deb"
dpkg -i /tmp/chef_12.21.14-1_amd64.deb
git config --global push.default simple

echo "Host github.com" >> ~/.ssh/config
echo "  Compression yes" >> ~/.ssh/config

update-alternatives --config editor
