package cheshire


import (
    "bufio"
    "fmt"
    "log"
    "net"
    "sync"
    "io"
        // 
)

type BinaryWriter struct {
    serverConfig *ServerConfig
    conn         net.Conn
    writerLock   sync.Mutex
}

func (this *BinaryWriter) Write(response *Response) (int, error) {
    defer this.writerLock.Unlock()
    this.writerLock.Lock()
    bytes, err := BIN.WriteResponse(response, this.conn)
    return bytes, err
}

func (this *BinaryWriter) Type() string {
    return BIN.Type()
}

func BinaryListen(port int, config *ServerConfig) error {
    ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    defer ln.Close()
    if err != nil {
        // handle error
        log.Println(err)
        return err
    }
    log.Println("Binary Listener on port: ", port)
    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Print(err)
            // handle error
            continue
        }
        go handleConnection(&BinaryWriter{serverConfig: config, conn: conn})
    }
    return nil
}


func handleConnection(conn *BinaryWriter) {
    defer conn.conn.Close()
    // log.Print("CONNECT!")

    decoder := BIN.NewDecoder(bufio.NewReader(conn.conn))

    for {
        req, err := decoder.DecodeRequest()
        if err == io.EOF {
            log.Print(err)
            break
        } else if err != nil {
            log.Print(err)
            break
        }
        // //request
        controller := conn.serverConfig.Router.Match(req.Method(), req.Uri())
        go HandleRequest(req, conn, controller, conn.serverConfig)
    }

    log.Print("DISCONNECT!")
}
