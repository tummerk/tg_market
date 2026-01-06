package server

// Данный сервер просто объединяет специфичные HTTP сервера, отвечающие за обработку конкретных сущностей
// В примере у нас есть только ExampleServer, но их может быть несколько
type Server struct {
	ExampleServer
}

func NewServer(
	exampleServer ExampleServer,
) Server {
	return Server{
		ExampleServer: exampleServer,
	}
}
