package main

import (
    "crypto/md5"
    "flag"
    "fmt"
    "html/template"
    "io"
    "log"
    "net/http"
    "os"
    "path"
    "strconv"
    "strings"
    "sync"
    "time"
    // "html"
)

var (
    tmpl    = template.Must(template.ParseFiles("drop_form.html"))
    dropDir string
    agents  agentdata = agentdata{Data: make(map[string]*agent)}
)

type FormData struct {
    SID string
}

type agent struct {
    Seen      time.Time
    SigExpire time.Time
    SigCnt    int
}

type agentdata struct {
    Data map[string]*agent
    Mux  sync.Mutex
}

// Mux usage--
// NetPrefixes.Mux.Lock()
// NetPrefixes.Data[prefix] = struct{}{}
// NetPrefixes.Mux.Unlock()

func genToken() string {
    crutime := time.Now().Unix()
    h := md5.New()
    io.WriteString(h, strconv.FormatInt(crutime, 10))
    token := fmt.Sprintf("%x", h.Sum(nil))
    return token
}

func genHash(f string) string {
    data, err := os.Open(f)
    if err != nil {
        log.Fatal(err)
    }
    defer data.Close()

    h := md5.New()
    if _, err := io.Copy(h, data); err != nil {
        log.Fatal(err)
    }
    return fmt.Sprintf("%x", h.Sum(nil))
}

//func manageDrops(t <-chan time.Time)() {
// range over t to perform routine checks on dir contents and file inventory
//for now := range t {
// status := fmt.Sprintf("Ran manageDrops function body at: %v", now)
//log.Print(status)
//}
//}

func fileServerWithLogging(fs http.FileSystem) http.Handler {
    fsh := http.FileServer(fs)
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        client := strings.Split(r.RemoteAddr, ":")[0]

        // track all clients
        if _, ok := agents.Data[client]; !ok {
            agents.Data[client] = &agent{
                Seen:      time.Now(),
                SigCnt:    0,
                SigExpire: time.Now().Add(300 * time.Second),
            }
        } else {
            if time.Now().After(agents.Data[client].SigExpire) {
                log.Println("Expired signal session with ", client)
                agents.Data[client].SigCnt = 0
                agents.Data[client].SigExpire = time.Now().Add(300 * time.Second)
            }
        }

        switch r.URL.Path {
        case "/drop":
            switch r.Method {
            case "GET":
                data := FormData{SID: genToken()}
                err := tmpl.Execute(w, data)
                if err != nil {
                    log.Print(err)
                }
                log.Printf("%s accessed the drop form", client)
            case "POST":
                // take an upload as form-data
                r.ParseMultipartForm(32 << 20)

                // Access the drops key which is a list of uploaded files
                fhs := r.MultipartForm.File["drops"]
                log.Printf("Recieving file drop from %s", client)
                for _, fh := range fhs {
                    // open a file handle from tmp or cache
                    f, err := fh.Open()
                    if err != nil {
                        log.Print(err)
                    }
                    defer f.Close()
                    // open a file handle for the destination file
                    out, err := os.OpenFile(dropDir+"/"+fh.Filename, os.O_WRONLY|os.O_CREATE, 0666)
                    if err != nil {
                        log.Print(err)
                    }
                    defer out.Close()
                    // copy the reader to the writer
                    io.Copy(out, f)
                    log.Printf("%s dropped file %s", client, fh.Filename)
                }
                http.Redirect(w, r, "/", http.StatusAccepted)
            }
        case "/d":
            switch r.Method {
            case "POST":
                // take an upload as form-data
                r.ParseMultipartForm(32 << 20)

                // Access the drops key which is a list of uploaded files
                fhs := r.MultipartForm.File["d"]
                log.Printf("Recieving file drop from %s", client)
                for _, fh := range fhs {
                    // open a file handle from tmp or cache
                    f, err := fh.Open()
                    if err != nil {
                        log.Print(err)
                    }
                    defer f.Close()
                    // open a file handle for the destination file
                    out, err := os.OpenFile(dropDir+"/"+fh.Filename, os.O_WRONLY|os.O_CREATE, 0666)
                    if err != nil {
                        log.Print(err)
                    }
                    defer out.Close()
                    // copy the reader to the writer
                    io.Copy(out, f)
                    log.Printf("%s dropped file %s", client, fh.Filename)
                }
                w.WriteHeader(http.StatusAccepted)
            }
        case "/signal":
            // they haven't asked enough
            if agents.Data[client].SigCnt < 4 {
                agents.Data[client].SigCnt++
                w.WriteHeader(http.StatusUnauthorized)
                w.Write([]byte("Unauthorized"))

            } else if agents.Data[client].SigCnt == 4 {
                // givem the signal
                w.WriteHeader(http.StatusAccepted)
                w.Write([]byte("listdir"))
            } else {
                // Something went wrong, restart signal tracking
                agents.Data[client].SigCnt = 0
                agents.Data[client].SigExpire = time.Now().Add(300 * time.Second)
                w.WriteHeader(http.StatusUnauthorized)
                w.Write([]byte("Unauthorized"))
            }
        case "/":
            fsh.ServeHTTP(w, r)
            log.Printf("%s accessed the dropsite file server", client)
        default:
            if _, err := os.Stat(dropDir + path.Clean(r.URL.Path)); err != nil {
                if os.IsNotExist(err) {
                    log.Printf("%s requested non-existent resource %s", client, path.Clean(r.URL.Path))
                    http.Redirect(w, r, "/", http.StatusNotFound)
                }
            } else {
                http.ServeFile(w, r, dropDir+path.Clean(r.URL.Path))
                log.Printf("%s retrieved file %s", client, path.Clean(r.URL.Path))
            }
        }
    })
}

func main() {
    // t := time.Tick(time.Minute / 2)
    // go manageDrops(t)

    flag.StringVar(&dropDir, "dir", "/var/dropsite", "Directory to store files.")
    cert_pem := flag.String("cert", "cert.pem", "Server TLS certificate.")
    key_pem := flag.String("key", "key.pem", "Server TLS certificate key.")
    httpPort := flag.String("http_port", "8880", "Port for HTTP dropsite.")
    httpsPort := flag.String("https_port", "8443", "Port for HTTPS dropsite.")

    flag.Parse()

    log.Printf("Running dropsite on ports %s and %s. Drop directory %s", *httpPort, *httpsPort, dropDir)

    go http.ListenAndServe(":"+*httpPort, fileServerWithLogging(http.Dir(dropDir)))
    werr := http.ListenAndServeTLS(":"+*httpsPort, *cert_pem, *key_pem, fileServerWithLogging(http.Dir(dropDir)))
    if werr != nil {
        log.Fatal(werr)
    }
}
