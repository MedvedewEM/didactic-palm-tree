package models

type Server struct {
	ID   int
	Host string
}

type FilePartServer struct {
	ID   int
	Host string
	PartSize int
}

func ServerToServerIDs(servers []Server) []int {
	serverIDs := make([]int, len(servers))
	for i, server := range servers {
		serverIDs[i] = server.ID
	}

	return serverIDs
}