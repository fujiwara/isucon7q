version = "v0.8.0"
arch = node[:kernel][:machine] == "x86_64" ? "amd64" : "386"
filename = "stretcher-%s-linux-%s" % [version, arch]
download_url = "https://github.com/fujiwara/stretcher/releases/download/%s/%s.zip" % [version, filename]

remote_file "/tmp/#{filename}.zip" do
  owner "root"
  group "root"
  mode  "644"
  source download_url
  not_if "/usr/local/bin/stretcher -v | grep 'version: #{version}'"
  notifies :run, "bash[install stretcher]"
end

bash "install stretcher" do
  cwd "/tmp"
  user "root"
  action :nothing
  code <<-EOF
    unzip #{filename}.zip
    install #{filename} /usr/local/bin/stretcher
  EOF
end
