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

func ServersToServerIDs(servers []Server) []int {
	serverIDs := make([]int, len(servers))
	for i, server := range servers {
		serverIDs[i] = server.ID
	}

	return serverIDs
}

func ServersToHosts(servers []Server) []string {
	serverHosts := make([]string, len(servers))
	for i, server := range servers {
		serverHosts[i] = server.Host
	}

	return serverHosts
}

func FilePartServersToServerIDs(servers []FilePartServer) []int {
	serverIDs := make([]int, len(servers))
	for i, server := range servers {
		serverIDs[i] = server.ID
	}

	return serverIDs
}