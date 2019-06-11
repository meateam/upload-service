package main

import "github.com/meateam/upload-service/server"

func main() {
	server.NewServer().Serve(nil)
}
