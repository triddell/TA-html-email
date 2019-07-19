.PHONY : default
default:
	GOOS=darwin GOARCH=amd64 go build -o darwin_x86_64/bin/html-email
	GOOS=linux GOARCH=amd64 go build -o linux_x86_64/bin/html-email
	GOOS=windows GOARCH=amd64 go build -o windows_x86_64/bin/html-email.exe