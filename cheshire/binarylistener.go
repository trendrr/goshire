package cheshire


import (
    "bufio"
    "encoding/binary"
    "fmt"
    "io"
    "log"
    "net"
    "sync"
        "github.com/trendrr/goshire/dynmap"
)

type BinaryWriter struct {
    serverConfig *ServerConfig
    conn         net.Conn
    writerLock   sync.Mutex
}

func (this *BinaryWriter) Write(response *Response) (int, error) {
    return 0, nil
    // json, err := json.Marshal(response)
    // if err != nil {
    //     return 0, err
    // }
    // defer this.writerLock.Unlock()
    // this.writerLock.Lock()
    // bytes, err := this.conn.Write(json)
    // return bytes, err
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

// Reads a length prefixed byte array
// it assumes the first 
func readByteArray(reader io.Reader) ([]byte, error) {
    length := int16(0)
    err := binary.Read(reader, binary.BigEndian, &length)
    if err != nil {
        return nil, err
    }

    bytes := make([]byte, length)
    _, err = io.ReadAtLeast(reader, bytes, int(length))
    return bytes, err
}

//Reads a length prefixed utf8 string 
func readString(reader io.Reader) (string, error) {
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
    //gzip, ect..  content will always be a byte array
    contentEncoding int8
    contentLength int64
}

func handleConnection(conn *BinaryWriter) {
    defer conn.conn.Close()
    // log.Print("CONNECT!")

    reader := bufio.NewReader(conn.conn)

    //read the hello
    hello, err := readString(reader)
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
        if err != nil {
            log.Print(err)
            break
        }
        //TODO Decode the params!
        log.Println(params)

        
        err = binary.Read(reader, binary.BigEndian, &h.contentEncoding)
        if err != nil {
            log.Print(err)
            break
        }

        err = binary.Read(reader, binary.BigEndian, &h.contentLength)
        if err != nil {
            log.Print(err)
            break
        }
        log.Println(h)
        // var req Request
        // err = dec.Decode(&req)

        // if err == io.EOF {
        //     log.Print(err)
        //     break
        // } else if err != nil {
        //     log.Print(err)
        //     break
        // }
        // //request
        // controller := conn.serverConfig.Router.Match(req.Method(), req.Uri())
        // go HandleRequest(&req, conn, controller, conn.serverConfig)
    }

    log.Print("DISCONNECT!")
}
