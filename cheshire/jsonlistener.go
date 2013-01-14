package cheshire

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"net"
)

type JsonConnection struct {
	serverConfig *ServerConfig
	conn         net.Conn
	writer       *bufio.Writer
	writerLock sync.Mutex
}

func (conn JsonConnection) Write(response *Response) (int, error) {
	json, err := json.Marshal(response)
	if err != nil {
		//TODO: uhh, do something..
		log.Print(err)
	}
	defer conn.writerLock.Unlock()
	conn.writerLock.Lock()
	// log.Println("writing ", string(json))
	bytes, err := conn.writer.Write(json)
	conn.writer.Flush()
	return bytes, err
}

func JsonListen(port int, config *ServerConfig) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
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
		go handleConnection(JsonConnection{serverConfig: config, conn: conn})
	}
	return nil
}

func handleConnection(conn JsonConnection) {
	defer conn.conn.Close()
	// log.Print("CONNECT!")
	conn.writer = bufio.NewWriter(conn.conn)

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

		controller := conn.serverConfig.Router.Match(req.Strest.Method, req.Strest.Uri)
		go controller.HandleRequest(&req, conn)
	}
	log.Print("DISCONNECT!")
}
