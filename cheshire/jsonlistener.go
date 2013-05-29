package cheshire

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

type JsonWriter struct {
	serverConfig *ServerConfig
	conn         net.Conn
	writerLock   sync.Mutex
}

func (this *JsonWriter) Write(response *Response) (int, error) {
	json, err := json.Marshal(response)
	if err != nil {
		return 0, err
	}
	defer this.writerLock.Unlock()
	this.writerLock.Lock()
	bytes, err := this.conn.Write(json)
	return bytes, err
}

func (this *JsonWriter) Type() string {
	return "json"
}

func JsonListen(port int, config *ServerConfig) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	defer ln.Close()
	if err != nil {
		// handle error
		log.Println(err)
		return err
	}
	log.Println("Json Listener on port: ", port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Print(err)
			// handle error
			continue
		}
		go handleConnection(&JsonWriter{serverConfig: config, conn: conn})
	}
	return nil
}

func handleConnection(conn *JsonWriter) {
	defer conn.conn.Close()
	// log.Print("CONNECT!")

	dec := json.NewDecoder(bufio.NewReader(conn.conn))

	for {
		var req Request
		err := dec.Decode(&req)

		if err == io.EOF {
			log.Print(err)
			break
		} else if err != nil {
			log.Print(err)
			break
		}
		//request
		controller := conn.serverConfig.Router.Match(req.Method(), req.Uri())
		go HandleRequest(&req, conn, controller, conn.serverConfig)
	}

	log.Print("DISCONNECT!")
}
