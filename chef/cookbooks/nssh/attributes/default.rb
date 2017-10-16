default[:nssh][:version] = "v0.0.2"
default[:nssh][:arch] = node[:kernel][:machine] == "x86_64" ? "amd64" : "386"
