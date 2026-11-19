# Spread love

via QR codes.

## Setup

Create a systemd service on your VPS:

```ini
[Unit]
Description=Spreadlove main service
After=network.target

[Service]
Type=simple
WorkingDirectory=/home/<VPS_USER>/spreadlove
ExecStart=/home/<VPS_USER>/spreadlove/bin/spreadlove
User=<VPS_USER>
Group=<VPS_USER>
Restart=always

[Install]
WantedBy=multi-user.target
```

Start:

```sh
sudo systemctl daemon-reload
sudo systemctl enable spreadlove.service --now
```

Also ensure that you can execute `systemctl restart spreadlove.service` without SUDO:

1. `sudo EDITOR=vim visudo`
2. vps_user ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart spreadlove.service

This is needed for the final step of deploy, where we restart the systemd service.

You will also need to add necessary secrets to GH secrets. Better to generate a new public/private pairing for the rsync
transfer, since it cannot be password protected:
<https://github.com/easingthemes/ssh-deploy?tab=readme-ov-file#configuration>

## Email setup

- [x] DNS
- [x] Program to write email
- [x] Systemd timer to email every day

First, create a similar systemd service to the one above for `cmd/mailer/main.go`. Change `Type=oneshot` and don't set
`Restart` directive.

Then, create a systemd timer:

```ini
[Unit]
Description=Run mailer every day at 13

[Timer]
OnCalendar=*-*-* 13:00:00
Persistent=true

[Install]
WantedBy=timers.target
```

## TODO

- [x] submit message
- [x] admin panel
- [x] ability to mark pending messages as qualified to move into actual messages
- [x] QR code generation
- [x] printing
- [x] email with pending messages
