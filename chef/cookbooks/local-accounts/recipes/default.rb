[
  'ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA7MSS+wmwvv0Dj0eH7ow39RoeU04P4MIRcsz1c7UMtglJAUjTGI6vAkdPxXJhoBfDUZ+XvRF7ZOOYtaBThFqkeg0w5oOjhFY7r8VuBOt2okqyyDG03fzxtlihywWdXnU5GiLZvoXskT5XsaqM1heHcuGMRMeDktTKg0JxY2RYwl2Rns0eQLo8yz3vluRrGzKmxtJQtNBl9XFz0d+PS2dzVobfjtBhliwwJ4aLxj8Xpsa94eLoCf7yKp0pk+aYVc9rX6ctvuvza1o2xpFqgXojJV5Rt/as5uLW61zi+mowk0T9IpzuneSiM8i+CQis/9qHrorHUr0IE628z7nT37qAbw==',
  'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDmfejyq7VNaHj3YfRO9iZtC/6p0f7J5xgdvATTNCk7Dw+y1KS7JbVsDo2Sbztm0QmoA2cyWiMkMKQshqNZcTpL0yfAEmSV2/HdzVkfXL1dbcPMXOZ6xLcXXH1MO18SWcPNKcv/s4yAvgsjSA1ji83GWCOPQoktkoY23QxSxNvsTeYKIT1dsjcpOFcO63aLQ54KD/wt62FGw5kzxf+9NSFr1UUqNebK7GhJRKaxb2QtSW0T3z6HthQc5ehuWWJOXZ6TMqoFzrlgcjWFEk/56EcfGc//CgBY/NgAJJ264ylmyca4ppzRsNGCISrRmc5nZ5ufOoSONw9JguTxIFvwgwsP',
  'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDHzuBRzL1li7a1icUz+je46TIQY02R2EO5gtqxUx5Gu+pWVCmcifFkDyotNfe7WycFdfZlQ11Uc3vox4qggTf8IRBPH24K5PAmwsYsACm/IoptAEOxwtZcesNu+uirhKPPTtCUa95MWSSSJZ9mGvFXXGHzTc8ce7TFx/95UOcNG98v8ypMWpSBBDMZNF1XWvglJW7tl0IFX3d9QNWnjMvufPLStqUtsLTty4eZPedrIelRL1dOzFFILsKn3tlg0tZj5V0BOQ7i9c3IOYZSYmeCLx7w7xbF84QjnUHMmBVNXdO/Yq2UE+Z93RBmRqkUNZCqWb7Wgsi+/Ed2eElxVkrF',
  'ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAxMmtxtTvqFo8YFXLYcBk9oi9ZUbcf/Yrj4VK00MP/sBrc8YfJ1gAp9DTkemR1JcbjOF/nhDjw14mvK8KPYWqmCVzbpL0SummUF987oI8pfZWCh7mnFmSEIwmwZ8nyYVLw1YQk50Z84oRRa/Q1C5q0Tft9wC7xYSDfSY5Hrft0hhqJexYCdfAQ5nnwx3377tCCgRleEgBS0jBcd0qwuyu3b1b8G7yFfZPOFNrzJss0jYY8X2zec31qdh2p3n2P38lRjfSKbjamXZWnEsFf4ju6NcpNUfFcGqtImkobIR/dDlSjvR/BD48xNDt9bFjqzNF1t8qBEqNqsC695DDdbQS9Q==',
  'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7joI29NVJhWFEG2rRgJRD4MVK1/NutqxWxLMaBuq+545xckqhNOP8LdpV+3jaRm66M5xSoKd2mEj//LLquxknPFn6ggUqUiB08yDFRlqL7CA3N36HcrAZ4oiVHuHsq1mlcYIalVlnl8X0BeVDo7JUabOjSrAd4izhnGvafIGRzhiSe5TS6uedWCggj7fBHp3FjVbKoYYJPAUQpX+Sex4go/a12aDhCimEnMsF3Wl+p19MpRsraIvCmq5AIpzZHUa9S4iFRFnAFtiiHkHUUjx/zOZfXgh1/64fY73ytoJyToGniwjBKyxmdoL1nSKKjmfLr1pkgny+BKcc+dZvrqmb',
].each do |pubkey|
  bash "update-pubkey #{pubkey[0..63]}" do
    user "isucon"
    code "echo '#{pubkey}' >> /home/isucon/.ssh/authorized_keys"
    not_if "fgrep '#{pubkey}' /home/isucon/.ssh/authorized_keys"
  end
end

bash "update bashrc" do
  user "isucon"
  code "echo 'export NOPASTE=https://example.com/np' >> /home/isucon/.bashrc"
  not_if "fgrep NOPASTE /home/isucon/.bashrc"
end

include_recipe 'local-accounts::fujiwara'
include_recipe 'local-accounts::acidlemon'
include_recipe 'local-accounts::handlename'

group 'sudo' do
  action [:modify]
  members ['fujiwara', 'acidlemon', 'handlename']
  append true
end
