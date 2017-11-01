package main

import (
	"encoding/json"
	"log"
	"net/http"
	"gopkg.in/mux"
	"gopkg.in/mgo.v2"
	"time"
	"regexp"
	//"gopkg.in/mgo.v2/bson"
	"strings"
	"gopkg.in/mgo.v2/bson"
	"math/rand"
	"fmt"
	rand2 "crypto/rand"

	"strconv"
	"io"
)

/*
adddd
func ErrorWithJSON(w http.ResponseWriter, json []byte, code int) {
	var uuid, _ = newUUID()
	jobid := strconv.Itoa(randInt())

	w.Header().Set("x-request-id", uuid)
	w.Header().Set("datetime", time.Now().Format("2006-01-02 15:04:05+0700"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("x-roundtrip", "")
	w.Header().Set("x-job-id", jobid)

	//w.WriteHeader(code)
	//fmt.Fprintf(w, "{message: %q}", message)
	w.Write(json)
}
*/

func ResponseWithJSON(w http.ResponseWriter, json []byte, code int) {
	var uuid, _ = newUUID()
	jobid := strconv.Itoa(randInt())

	w.Header().Set("x-request-id", uuid)
	w.Header().Set("datetime", time.Now().Format("2006-01-02 15:04:05+0700"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("x-roundtrip", "")
	w.Header().Set("x-job-id", jobid)
	w.WriteHeader(code)
	//w.Write(json)
	w.Write(json)
}

type WalletAccount struct {
	ID        bson.ObjectId `bson:"_id,omitempty"`
	CitizenID     int     `json:"citizen_id" bson:"citizen_id"`
	FullName      string  `json:"full_name" bson:"full_Name"`
	WalletID      int     `json:"wallet_id" bson:"wallet_id"`
	OpenDateTime  string  `json:"open_datetime" bson:"open_datetime"`
	LedgerBalance float32 `json:"ledger_balance" bson:"ledger_balance"`
}

type MsgBodySuccess struct {
	RsBody RsBody `json:"rsBody"`
	//Error ErrorList `json:"error"`
}

type MsgBodyError struct {
	//RsBody RsBody `json:"rsBody"`
	Error ErrorList `json:"error"`
}

type RsBody struct {
	WalletID	int		`json:"wallet_id"`
	OpenDateTime  string  `json:"open_datetime"`
}

type Error struct {
	ErCode string 	`bson:"error code"`
	ErDesc string	`bson:"error description"`
}

type ErrorList struct {
	Error []Error
}

func main() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	ensureIndex(session)

	mux := mux.NewRouter()
	mux.HandleFunc("/v1/accounts/{wallet_id}", getAccountByWalletID(session)).Methods("GET")
	//mux.HandleFunc("/v1/accounts/search", getAccountByFullName(session)).Methods("GET")
	//mux.HandleFunc("/v1/accounts", getAccountByCitizenID(session)).Methods("GET")
	mux.HandleFunc("/v1/accounts", createWallets(session)).Methods("POST")
	//http.ListenAndServe("localhost:5000", mux)
	log.Fatal(http.ListenAndServe("localhost:3334", mux))
}

func ensureIndex(s *mgo.Session) {
	session := s.Copy()
	defer session.Close()

	c := session.DB("wallets").C("accounts")

	index := mgo.Index{
		Key:		[]string{"citizen_id"},
		Unique:		true,
		DropDups:	true,
		Background:	true,
		Sparse:		true,
	}

	err := c.EnsureIndex(index)
	if err != nil {
		panic(err)
	}
}

func createWallets(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		var accounts WalletAccount
		var errorlist ErrorList

		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&accounts)
		if err != nil {
			errorlist.Error = append(errorlist.Error,Error{"999", "Incorrect Body"})
		}
		if !LenCitizenId(accounts.CitizenID) {
			errorlist.Error = append(errorlist.Error,Error{"001", "Incorrect Citizen ID"})
		}

		if (!IsLetter(accounts.FullName)) || (!Len(accounts.FullName)) {
			errorlist.Error = append(errorlist.Error,Error{"003", "Incorrect Name"})
		}

		c := session.DB("wallets").C("accounts")
		currentid,err:= c.Count()
		wwid, _ :=strconv.Atoi(generateWalletID(currentid))
		accounts.FullName=strings.ToUpper(accounts.FullName)
		accounts.WalletID=wwid
		accounts.OpenDateTime = time.Now().Format("2006-01-02 15:04:05 GMT+0700")
		accounts.LedgerBalance = 0.00

		if len(errorlist.Error)==0 {
			err = c.Insert(accounts)
			if err != nil {
				if mgo.IsDup(err) {
					errorlist.Error = append(errorlist.Error, Error{"002", "Duplicate Citizen ID"})
				}
			}
		}
		msgbodysuccess :=MsgBodySuccess{}
		msgbodyer :=MsgBodyError{}
		if len(errorlist.Error)==0 {
			respbody := RsBody{
				OpenDateTime:accounts.OpenDateTime,
				WalletID:accounts.WalletID,
			}
			msgbodysuccess.RsBody=respbody
			respBody, err := json.MarshalIndent(msgbodysuccess, "", "  ")
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Success")
			ResponseWithJSON(w, respBody, http.StatusCreated)

		} else {
			msgbodyer.Error = errorlist
			respBody, err := json.MarshalIndent(msgbodyer, "", "  ")
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Error")
			ResponseWithJSON(w, respBody, http.StatusBadRequest)
		}

	}
}


func getAccountByWalletID(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	/*
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		vars := mux.Vars(r)
		wallets := vars["wallet_id"]

		c := session.DB("wallets").C("accounts")

		var accounts WalletAccount
		//var errorlist ErrorList
		err := c.Find(bson.M{"wallet_id": wallets}).One(&accounts)

		/*
		if err != nil {
			errorlist.Error = append(errorlist.Error,Error{"003", "Database error"})
			log.Println("Failed find book: ", err)
			return
		}

		The zero values for integer and floats is 0. nil is not a valid integer or float value.
		A pointer to an integer or a float can be nil, but not its value.


		var intPointer *int
		intValue := accounts.WalletID
		intPointer = &intValue

		if intPointer == nil {
			errorlist.Error = append(errorlist.Error,Error{"003", "Incorrect Name"})
			return
		}

		respBody, err := json.MarshalIndent(accounts, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		ResponseWithJSON(w, respBody, http.StatusOK)
	}
	*/
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		vars := mux.Vars(r)
		wallets := vars["wallet_id"]

		c := session.DB("wallets").C("accounts")
		var accounts WalletAccount
		err := c.Find(bson.M{"wallet_id": wallets}).One(&accounts)

		respBody, err := json.MarshalIndent(accounts, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		ResponseWithJSON(w, respBody, http.StatusOK)
	}
}


func getAccountByFullName(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		vars := mux.Vars(r)
		wallets := vars["wallet_id"]

		c := session.DB("wallets").C("accounts")

		var accounts WalletAccount
		err := c.Find(bson.M{"wallet_id": wallets}).One(&accounts)

		/*
				if err != nil {
					ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
					log.Println("Failed find wallet_id: ", err)
					return
				}

				if accounts.WalletID == nil {
					ErrorWithJSON(w, "Book not found", http.StatusNotFound)
					return
				}
		*/
		respBody, err := json.MarshalIndent(accounts, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		ResponseWithJSON(w, respBody, http.StatusOK)
	}
}

func getAccountByCitizenID(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		vars := mux.Vars(r)
		wallets := vars["wallet_id"]

		c := session.DB("wallets").C("accounts")

		var accounts WalletAccount
		err := c.Find(bson.M{"wallet_id": wallets}).One(&accounts)

		/*
				if err != nil {
					ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
					log.Println("Failed find wallet_id: ", err)
					return
				}

				if accounts.WalletID == nil {
					ErrorWithJSON(w, "Book not found", http.StatusNotFound)
					return
				}
		*/
		respBody, err := json.MarshalIndent(accounts, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		ResponseWithJSON(w, respBody, http.StatusOK)
	}
}
var IsLetter = regexp.MustCompile(`^[a-zA-Z.,-]+( [a-zA-Z.,-]+)+$`).MatchString

func Len(s string) bool {
	if len(s)<=50 {
		return true
	}
	return false
}

func LenCitizenId(i int) bool {
	if len(strconv.Itoa(i))==13 {
		return true
	}
	return false
}

func generateWalletID(currentid int) string {
	currentid++
	wall := fmt.Sprintf("%010d", currentid)
	sum:=0
	for v,i := range(wall) {
		intValue, _ := strconv.Atoi(string(i))
		//fmt.Println("--------------------------------------------------")
		//fmt.Println(intValue + " " + v+1 )
		sum+=intValue*int(v+2)

	}

	sum=sum%10
	return "1"+wall+strconv.Itoa(sum)
}

func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand2.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func randInt() (int) {
	return rand.Intn(9999999999)
}






