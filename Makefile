buildlinux:
	OARCH=arm64 GOOS=linux go build -o homelab_agent main.go