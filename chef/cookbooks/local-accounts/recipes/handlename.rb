user "handlename" do
  home "/home/handlename"
  manage_home true
  shell "/bin/bash"
end

directory "/home/handlename/.ssh" do
  mode 0700
  owner "handlename"
end

remote_file "/home/handlename/.ssh/authorized_keys" do
  mode 0600
  owner "handlename"
  source "https://github.com/handlename.keys"
  action :create_if_missing
end
