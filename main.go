package main

import (
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	base64 "encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"

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

func createBurnmsg(msg string) string {
	cipherKey := []byte(os.Getenv("SECRET_KEY"))
	block, blockerr := aes.NewCipher(cipherKey)
	if blockerr != nil {
		panic(blockerr.Error())
	}
	plainText := []byte(msg)
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
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
	w.Header().Set("Content-Type", "application/json")
	res := readAndBurn(ps.ByName("id"))
	if res == "nan" {
		w.WriteHeader(http.StatusNotFound)
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

func handleBurnmsg(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {

	type message_struct struct {
		Message string `json:"message"`
	}
	decoder := json.NewDecoder(r.Body)
	var m message_struct
	err2 := decoder.Decode(&m)
	if err2 != nil {
		panic(err2)
	}

	resp := make(map[string]string)
	create := createBurnmsg(m.Message)
	resp["msgId"] = create
	log.Println("Burnmessage " + create + " inserted into the database")
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	rw.Write(jsonResp)
}

func middleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_HEADER"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE")
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

func main() {
	router := httprouter.New()
	router.GET("/:id", middleware(getBurnmsg))
	router.POST("/", middleware(handleBurnmsg))

	http.ListenAndServe(":8080", router)
}
