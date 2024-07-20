package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var addr = flag.String("addr", "localhost:8081", "Listening address")

var db = sqlx.MustConnect("sqlite3", "pipi_pupu.sqlite3")

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
func KraftSendError(w http.ResponseWriter, errStr string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	json.NewEncoder(w).Encode(ApiError{
		Err: errStr,
	})
}

func KraftStopVM(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	vmName := r.PathValue("vm_name")

	var id string
	err := db.Get(&id, `SELECT id FROM name_to_id WHERE name = ?`, vmName)
	if err != nil {
		KraftSendError(w, "VM not found", http.StatusNotFound)
		return
	}

	log.Printf("Starting %s", id)
	err = exec.Command("kraft", "cloud", "instance", "stop", id).Run()

	if err != nil {
		KraftSendError(w, "Failed to start VM", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type ApiError struct {
	Err string `json:"err"`
}

func GenerateRandomHex(length int) (string, error) {
	// Each byte is represented by two hex characters.
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type KraftAppInfo struct {
	AppArgs    string `json:"app_args"`
	KernelArgs string `json:"kernel_args"`
	Name       string `json:"name"`
	Size       string `json:"size"`
	Version    string `json:"version"`
}

type chanWriter struct {
	w  io.Writer
	ch chan []byte
}

func (e chanWriter) Write(p []byte) (int, error) {
	b := make([]byte, len(p))
	log.Printf("WE GUCCING ZA MESSAGE\n")
	copy(b, p)

	e.ch <- b

	return len(p), nil
}

func NewChanWriter() *chanWriter {
	return &chanWriter{
		ch: make(chan []byte, 128),
	}
}

type ChanInitMsg struct {
	id string
	ch *chanWriter
}

var chanChanHandler = make(chan ChanInitMsg)

func KraftUploadVM(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB max memory
	enableCors(&w)
	if err != nil {
		fmt.Fprintf(w, "Error parsing form: %v", err)
		return
	}

	file, handler, err := r.FormFile("vm")
	vmName := r.FormValue("name")
	tempDir := fmt.Sprintf("build-cache-%s", vmName)

	// Create tempfs folder
	for {
		err := os.Mkdir(tempDir, os.FileMode(0755))
		if os.IsExist(err) {
			os.RemoveAll(tempDir)
			continue
		} else if err != nil {
			fmt.Fprintf(w, "Error retrieving file: %v", err)
			return
		}
		break
	}

	if err != nil {
		fmt.Fprintf(w, "Error retrieving file: %v", err)
		return
	}
	defer file.Close()

	fmt.Fprintf(w, "Uploaded File: %+v\n", handler.Filename)
	fmt.Fprintf(w, "File Size: %+v\n", handler.Size)
	fmt.Fprintf(w, "MIME Header: %+v\n", handler.Header)

	w.WriteHeader(http.StatusOK)

	tempFile, err := GenerateRandomHex(32)
	if err != nil {
		KraftSendError(w, "Failed allocate project archive file", http.StatusBadRequest)
		return
	}

	outputZip, err := os.Create(fmt.Sprintf("%s/%s.zip", tempDir, tempFile))
	defer outputZip.Close()

	if err != nil {
		KraftSendError(w, "Failed allocate project archive file", http.StatusBadRequest)
		return
	}

	written, err := io.Copy(outputZip, file)

	if err != nil {
		KraftSendError(w, "Failed to write file", http.StatusBadRequest)
		return
	}

	cw := NewChanWriter()

	log.Printf("wrote %d for %s", written, vmName)

	chanChanHandler <- ChanInitMsg{
		id: vmName,
		ch: cw,
	}

	cmd := exec.Command("7z", "x", fmt.Sprintf("%s.zip", tempFile))
	cmd.Stdout = cw
	cmd.Stderr = cw
	cmd.Dir = tempDir

	err = cmd.Run()

	if err != nil {
		KraftSendError(w, "Invalid archive", http.StatusBadRequest)
		return
	}

	// kraft pkg --name index.unikraft.io/lbud/image:latest --push
	// kraft cloud instance create --start -p 443:8080 -M 1024  lbud/build-cache-mashina

	baseName := fmt.Sprintf("lbud/%s", vmName)
	imageName := fmt.Sprintf("index.unikraft.io/lbud/%s:latest", vmName)

	cmd = exec.Command("kraft", "pkg", "--name", imageName, "--push")
	cmd.Stdout = cw
	cmd.Stderr = cw
	cmd.Dir = tempDir
	err = cmd.Run()

	tries := 0
	gucci := false

	for {
		var images []KraftAppInfo
		data, err := exec.Command("kraft", "cloud", "image", "list", "-o", "json").Output()

		if err != nil {
			panic("bagamias")
		}

		err = json.Unmarshal(data, &images)

		for _, img := range images {
			if img.Name == baseName {
				gucci = true
			}
		}

		time.Sleep(100 * time.Millisecond)
		tries += 1

		if tries > 20 {
			gucci = true
		}

		if gucci {
			break
		}
	}

	cmd = exec.Command("kraft", "cloud", "instance", "create", "-M", "1024", baseName, "-o", "json")
	cmd.Dir = tempDir
	data, err := cmd.Output()

	var instanceInfo []KraftInstance
	err = json.Unmarshal(data, &instanceInfo)

	if err != nil {
		panic("RSALDJASLKDJ")
	}

	if len(instanceInfo) != 1 {
		panic("as203i1-")
	}

	db.MustExec(`INSERT INTO name_to_id (id, name) VALUES (?,?)`, instanceInfo[0].Name, vmName)

	return
}

type KraftInstance struct {
	AppExitCode     string `json:"app_exit_code"`
	Args            string `json:"args"`
	BootTime        string `json:"boot_time"`
	Created         string `json:"created"`
	Env             string `json:"env"`
	FQDN            string `json:"fqdn"`
	Image           string `json:"image"`
	Memory          string `json:"memory"`
	Name            string `json:"name"`
	NextRestart     string `json:"next_restart"`
	PrivateFQDN     string `json:"private_fqdn"`
	PrivateIP       string `json:"private_ip"`
	RestartAttempts string `json:"restart_attempts"`
	RestartCount    string `json:"restart_count"`
	RestartPolicy   string `json:"restart_policy"`
	ServiceGroup    string `json:"service_group"`
	StartCount      string `json:"start_count"`
	Started         string `json:"started"`
	State           string `json:"state"`
	StopOrigin      string `json:"stop_origin"`
	StopReason      string `json:"stop_reason"`
	Stopped         string `json:"stopped"`
	UpTime          string `json:"up_time"`
	UUID            string `json:"uuid"`
	Volumes         string `json:"volumes"`
}

func KraftStartVM(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	vmName := r.PathValue("vm_name")

	var id string
	err := db.Get(&id, `SELECT id FROM name_to_id WHERE name = ?`, vmName)
	if err != nil {
		KraftSendError(w, "VM not found", http.StatusNotFound)
		return
	}

	log.Printf("Starting %s", id)
	err = exec.Command("kraft", "cloud", "instance", "start", id).Run()

	if err != nil {
		KraftSendError(w, "Failed to start VM", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type VmList struct {
	Names []string `json:"names"`
}

func KraftListVM(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	var rows []string
	err := db.Select(&rows, `SELECT name FROM name_to_id`)

	if err != nil {
		panic("j1290djsok")
	}

	w.Header().Set("Content-Type", "application/json")

	fmt.Print(rows)
	json.NewEncoder(w).Encode(&VmList{Names: rows})
}

var upgrader = websocket.Upgrader{} // use default options
var chanMap = make(map[string]*chanWriter)

func main() {
	db.MustExec(`
	CREATE TABLE IF NOT EXISTS name_to_id (
		id   TEXT PRIMARY KEY,
		name TEXT
	);`)

	flag.Parse()

	api := http.NewServeMux()
	v1 := http.NewServeMux()
	vm := http.NewServeMux()
	ws := http.NewServeMux()

	ws.HandleFunc("/{id}/", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		log.Printf("HANDLIN %s", id)
		c, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Print("upgrade:", err)
			return
		}

		c.WriteMessage(1, []byte("Hello"))

		var ch *chanWriter

		for {
			cha, ok := chanMap[id]
			if ok {
				ch = cha
				break
			} else {
				log.Print("missing chan")
				time.Sleep(time.Second * 1)
			}
		}

		handler := func(c *websocket.Conn, ch *chanWriter) {
			defer c.Close()

			for {
				msg, ok := <-ch.ch

				if !ok {
					break
				}

				c.WriteMessage(1, msg)
			}

		}

		go handler(c, ch)
	})

	api.Handle("/api/", http.StripPrefix("/api", v1))

	v1.Handle("/v1/vm/", http.StripPrefix("/v1/vm", vm))
	v1.Handle("/v1/ws/", http.StripPrefix("/v1/ws", ws))

	vm.HandleFunc("POST /build", KraftUploadVM)
	vm.HandleFunc("POST /{vm_name}/start", KraftStartVM)
	vm.HandleFunc("POST /{vm_name}/stop", KraftStopVM)
	vm.HandleFunc("GET /list", KraftListVM)

	meme := func() {
		for {
			var msg ChanInitMsg = <-chanChanHandler
			log.Printf("Got chan for id %s\n", msg.id)
			chanMap[msg.id] = msg.ch
		}
	}

	go meme()

	server := http.Server{
		Addr:    "127.0.0.1:8081",
		Handler: api,
	}

	upgrader.CheckOrigin = func(_ *http.Request) bool {
		return true
	}

	log.Fatal(server.ListenAndServe())
}
