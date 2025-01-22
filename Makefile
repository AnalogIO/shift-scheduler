generate:
	echo "Generating..."
	cd src/tailwind && npm run build
	go run main.go generate 
