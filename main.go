package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	base64 "encoding/base64"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

func dbConn() (db *sql.DB) {
	dbDriver := "mysql"
	dbUser := os.Getenv("MYSQL_USERNAME")
	dbPass := os.Getenv("MYSQL_PASSWORD")
	dbName := os.Getenv("MYSQL_DATABASE")
	MYSQL_HOSTNAME := os.Getenv("MYSQL_HOSTNAME")
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@tcp"+"("+MYSQL_HOSTNAME+")"+"/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	return db
}

func bootstrapTable() {
	db := dbConn()

	_, err := db.Exec("CREATE TABLE IF NOT EXISTS burntable (id INT AUTO_INCREMENT PRIMARY KEY, messageId VARCHAR(255), messageEnc VARCHAR(255), messageIv VARCHAR(255))")
	if err != nil {
		panic(err)
	}

	defer db.Close()
}

func createBurnmsg(msg string) string {
	cipherKey := []byte(os.Getenv("SECRET_KEY"))
	block, blockerr := aes.NewCipher(cipherKey)
	if blockerr != nil {
		panic(blockerr.Error())
	}
	plainText := []byte(msg)
	cipherText := make([]byte, aes.BlockSize+len(plainText))

	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		log.Fatal(err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)

	messageIv := base64.RawStdEncoding.EncodeToString(iv)
	messageEnc := base64.RawStdEncoding.EncodeToString(cipherText)

	db := dbConn()
	id := uuid.New()
	insert, err := db.Prepare("INSERT INTO burntable (messageId, messageEnc, messageIv) VALUES(?,?,?)")
	if err != nil {
		panic(err.Error())
	}
	insert.Exec(id, messageEnc, messageIv)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	return id.String()
}

func readAndBurn(id string) string {
	cipherKey := []byte(os.Getenv("SECRET_KEY"))
	db := dbConn()

	getiv, err1 := db.Prepare("SELECT messageIv FROM burntable WHERE messageId=?")
	if err1 != nil {
		panic(err1.Error())

	}
	getenc, err := db.Prepare("SELECT messageEnc FROM burntable WHERE messageId=?")
	if err != nil {
		panic(err.Error())
	}

	var outiv string
	var outenc string
	err = getiv.QueryRow(id).Scan(&outiv)
	if err != nil {
		return "nan"
	}

	err = getenc.QueryRow(id).Scan(&outenc)
	if err != nil {
		panic(err.Error())
	}

	cipherText, err := base64.RawStdEncoding.DecodeString(outenc)
	if err != nil {
		panic(err.Error())
	}

	iv, err := base64.RawStdEncoding.DecodeString(outiv)
	if err != nil {
		panic(err.Error())
	}

	block, err := aes.NewCipher(cipherKey)
	if err != nil {
		panic(err.Error())
	}

	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	burnmessage, err := db.Prepare("DELETE FROM burntable WHERE messageId=?")
	if err != nil {
		panic(err.Error())
	}

	burnmessage.Exec(id)

	defer db.Close()
	return string(cipherText)
}

func getBurnmsg(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if len(ps.ByName("id")) >= 38 {
		resp := make(map[string]string)
		resp["error"] = "msgId length exceeded"
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			log.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResp)
		return
	}

	res := readAndBurn(ps.ByName("id"))
	if res == "nan" {
		w.WriteHeader(http.StatusOK)
		resp := make(map[string]string)
		resp["burnMsg"] = "Message does not exist or has been burned already"
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			log.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.Write(jsonResp)
		return
	}
	w.WriteHeader(http.StatusOK)
	resp := make(map[string]string)
	resp["burnMsg"] = res
	log.Println("Message " + ps.ByName("id") + " burned 🔥")
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	w.Write(jsonResp)
}

func handleBurnmsg(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	type message_struct struct {
		Message string `json:"message"`
	}
	decoder := json.NewDecoder(r.Body)
	var m message_struct

	err2 := decoder.Decode(&m)
	if err2 != nil {
		panic(err2)
	}

	if len(m.Message) >= 121 {
		resp := make(map[string]string)
		resp["error"] = "Message length exceeded"
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			log.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(jsonResp)
		return
	}

	resp := make(map[string]string)

	create := createBurnmsg(m.Message)
	resp["msgId"] = create
	log.Println("Burnmessage " + create + " inserted into the database")
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonResp)
}

func middleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if ps.ByName("id") == "ready" {
			readiness(w, r, ps)
			return
		}
		if ps.ByName("id") == "status" {
			liveness(w, r, ps)
			return
		}
		w.Header().Set("access-control-allow-headers", "Accept,content-type,Access-Control-Allow-Origin,access-control-allow-headers, access-control-allow-methods, Authorization")
		w.Header().Set("content-type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_HEADER"))
		w.Header().Set("access-control-allow-methods", "POST, GET, OPTIONS, DELETE")
		authHeader := r.Header.Get("Authorization")
		if authHeader != os.Getenv("AUTHHEADER_PASSWORD") {
			resp := make(map[string]string)
			resp["error"] = "Invalid or no credentials"
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				log.Fatalf("Error happened in JSON marshal. Err: %s", err)
			}
			w.WriteHeader(http.StatusForbidden)
			w.Write(jsonResp)
		} else {
			next(w, r, ps)
		}
	}
}

func readiness(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	conn := dbConn()
	err := conn.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	defer conn.Close()
}

func liveness(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	MYSQL_HOSTNAME := os.Getenv("MYSQL_HOSTNAME")
	timeout := 5 * time.Second

	conn, err := net.DialTimeout("tcp", MYSQL_HOSTNAME+":3306", timeout)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("MySQL server is not reachable"))
		return
	}
	defer conn.Close()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("MySQL server is reachable"))
}

func main() {
	bootstrapTable()

	router := httprouter.New()

	router.GET("/:id", middleware(getBurnmsg))
	router.POST("/", middleware(handleBurnmsg))

	http.ListenAndServe(":8080", router)
}
