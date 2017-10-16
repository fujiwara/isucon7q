user "acidlemon" do
  home "/home/acidlemon"
  supports :manage_home => true
  shell "/bin/bash"
end

directory "/home/acidlemon/.ssh" do
  mode 0700
  owner "acidlemon"
end

remote_file "/home/acidlemon/.ssh/authorized_keys" do
  mode 0600
  owner "acidlemon"
  source "https://github.com/acidlemon.keys"
  action :create_if_missing
end
