package goshire


import (
    "bufio"
    "encoding/binary"
    "fmt"
    "io"
    "log"
    "net"
    "sync"
)

type BinaryWriter struct {
    serverConfig *ServerConfig
    conn         net.Conn
    writerLock   sync.Mutex
}

func (this *BinaryWriter) Write(response *Response) (int, error) {
    json, err := json.Marshal(response)
    if err != nil {
        return 0, err
    }
    defer this.writerLock.Unlock()
    this.writerLock.Lock()
    bytes, err := this.conn.Write(json)
    return bytes, err
}

func (this *BinaryWriter) Type() string {
    return "bin"
}

func BinaryListen(port int, config *ServerConfig) error {
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
        go handleConnection(&BinaryWriter{serverConfig: config, conn: conn})
    }
    return nil
}

type header struct {
    headerLength int32
    txnId int64
    txnAccept int8
    method int8
    uri string
    paramEncoding int8
    params *dynmap.DynMap
    contentParamName string
    contentEncoding int8
    contentLength int64
}

// Reads a length prefixed byte array
// it assumes the first 
func readByteArray(reader *io.Reader) ([]byte, error) {
    length := int16(0)
    err = binary.Read(reader, binary.BigEndian, &length)
    if err != nil {
        return nil, err
    }

    bytes := make([]byte, length)
    _, err := io.ReadAtLeast(reader, bytes, length)
    return bytes, err
}

//Reads a length prefixed utf8 string 
func readString(reader *io.Reader) ([]byte, error) {
    b, err := readByteArray(reader)
    if err != nil {
        return "", err
    }
    return string(b), nil
}

type header struct {
    headerLength int32
    txnId int64
    txnAccept int8
    method int8
    uri string
    paramEncoding int8
    params *dynmap.DynMap
    contentParamName string
    contentEncoding int8
    contentLength int64
}

func handleConnection(conn *BinaryWriter) {
    defer conn.conn.Close()
    // log.Print("CONNECT!")

    reader := bufio.NewReader(conn.conn)

    //read the hello
    length := int16(0)
    hello, err = readString(reader)
    if err != nil {
        log.Print(err)
        //TODO: Send bad hello.
        return
    }
    log.Println(hello)


    for {

        h := &header{}
        err = binary.Read(reader, binary.BigEndian, &h.headerLength)
        if err != nil {
            log.Print(err)
            break
        }

        err = binary.Read(reader, binary.BigEndian, &h.txnId)
        if err != nil {
            log.Print(err)
            break
        }

        err = binary.Read(reader, binary.BigEndian, &h.txnId)
        if err != nil {
            log.Print(err)
            break
        }

        err = binary.Read(reader, binary.BigEndian, &h.method)
        if err != nil {
            log.Print(err)
            break
        }

        h.uri, err = readString(reader)
        if err != nil {
            log.Print(err)
            break
        }

        err = binary.Read(reader, binary.BigEndian, &h.paramEncoding)
        if err != nil {
            log.Print(err)
            break
        }

        params, err := readByteArray(reader)
        //TODO PArse the params!



        log.Println()

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
