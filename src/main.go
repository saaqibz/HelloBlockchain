package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Block struct {
	Index      int
	Timestamp  string
	BPM        int
	Hash       string
	PrevHash   string
	Difficulty int
	Nonce      string
}

type Message struct {
	BPM int
}

var Blockchain []Block
var mutex = &sync.Mutex{}
var difficulty = 2

func calculateHash(b *Block) string {
	msg := string(b.Index) + b.Timestamp + string(b.BPM) + b.PrevHash + b.Nonce + string(b.Difficulty)
	h := sha256.New()
	h.Write([]byte(msg))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func generateBlock(oldBlock Block, bpm int) (Block, error) {
	block := Block{
		Index:      oldBlock.Index + 1,
		Timestamp:  time.Now().String(),
		BPM:        bpm,
		PrevHash:   oldBlock.Hash,
		Difficulty: difficulty,
	}

	mineBlock(&block)

	block.Hash = calculateHash(&block)
	return block, nil
}

func mineBlock(b *Block) {
	for i := 0; ; i++ {
		b.Nonce = fmt.Sprintf("%d", i)
		b.Hash = calculateHash(b)
		if isHashValid(b) {
			log.Println("Mined a block!")
			return
		}
		time.Sleep(25 * time.Millisecond) // slow down the processing
		fmt.Printf(". ")
		if i%100 == 0 {
			log.Printf("Failed Attempt %s\n", spew.Sdump(b))
		}
	}
}

func isBlockValid(curr *Block, prev *Block) bool {
	return curr.Index == prev.Index+1 &&
		curr.PrevHash == prev.Hash &&
		calculateHash(curr) == curr.Hash
}

func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

// POW
func isHashValid(b *Block) bool {
	prefix := strings.Repeat("0", b.Difficulty)
	return strings.HasPrefix(b.Hash, prefix)
}

// Webserver
func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv("ADDR")
	log.Println("Listening on ", os.Getenv("ADDR"))
	s := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
	return muxRouter
}

func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(bytes))
}

func handleWriteBlock(w http.ResponseWriter, r *http.Request) {
	var m Message
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusInternalServerError, m)
		return
	}
	defer r.Body.Close()

	oldBlock := Blockchain[len(Blockchain)-1]
	mutex.Lock()
	newBlock, err := generateBlock(oldBlock, m.BPM)
	mutex.Unlock()
	if err != nil {
		respondWithJSON(w, r, http.StatusInternalServerError, m)
		return
	}
	if isBlockValid(&newBlock, &oldBlock) {
		newBlockchain := append(Blockchain, newBlock)
		Blockchain = newBlockchain
		spew.Dump(newBlockchain)
	} else {
		fmt.Println("Invalid Block: %s", spew.Sdump(newBlock))
		respondWithJSON(w, r, http.StatusBadRequest, m)
		return
	}

	respondWithJSON(w, r, http.StatusCreated, newBlock)
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		t := time.Now()
		genesisBlock := Block{
			Timestamp:  t.String(),
			Difficulty: difficulty,
		}
		spew.Dump(genesisBlock)
		mutex.Lock()
		Blockchain = append(Blockchain, genesisBlock)
		mutex.Unlock()
	}()
	log.Fatal(run())
}
