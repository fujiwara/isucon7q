filename = "nssh-%s-linux-%s" % [ node[:nssh][:version], node[:nssh][:arch] ]
download_url = "https://github.com/fujiwara/nssh/releases/download/%s/%s.zip" % [ node[:nssh][:version], filename ]

remote_file "/tmp/#{filename}.zip" do
  owner "root"
  group "root"
  mode "644"
  source download_url
  not_if "/usr/local/bin/nssh -v | grep 'version: #{node[:nssh][:version]}'"
  notifies :run, "bash[install nssh]"
end

bash "install nssh" do
  action :nothing
  cwd "/tmp"
  user "root"
  code <<-EOF
    unzip #{filename}.zip
    install #{filename} /usr/local/bin/nssh
  EOF
end
