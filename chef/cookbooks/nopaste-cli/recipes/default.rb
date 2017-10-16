#
# Cookbook Name:: nopaste-cli
# Recipe:: default
#
# Copyright 2016, YOUR_COMPANY_NAME
#
# All rights reserved - Do Not Redistribute
#

cookbook_file "/usr/local/bin/nopaste-cli" do
  owner "root"
  group "root"
  mode  "0755"
  source "nopaste-cli.amd64"
end
