package main

import "github.com/meateam/upload-service/server"

func main() {
	server.NewServer(nil).Serve(nil)
}
