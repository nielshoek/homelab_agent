buildlinux:
	OARCH=arm64 GOOS=linux go build -o homelab_agent main.go

copytopi:
	scp homelab_agent nielshoek@raspberrypi.local:/usr/local/bin/homelab_agent/homelab_agent 

# Movie path: /media/nielshoek/KINGSTON/jellyfin/media/Movies