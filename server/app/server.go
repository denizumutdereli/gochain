package app

import (
	"encoding/json"
	"io"
	"log"
	"main/block"
	"main/config"
	"main/utils"
	"main/wallet"
	"net/http"
	"strconv"
)

var cache map[string]*block.Blockchain = make(map[string]*block.Blockchain)

type BlockchainServer struct {
	port uint16
	conf *config.Blockchain
}

func NewBlockchainServer(port uint16, conf config.Blockchain) *BlockchainServer {
	return &BlockchainServer{
		port: port,
		conf: &conf,
	}
}

func (bcs *BlockchainServer) Port() uint16 { return bcs.port }

func (bcs *BlockchainServer) GetBlockchain() *block.Blockchain {
	bc, ok := cache["blockchain"]

	if !ok {
		minersWallet := wallet.NewWallet()
		bc, _ = block.CreateBlockchain(minersWallet.BlockchainAddress(), *bcs.conf)
		cache["blockchain"] = bc
		log.Printf("private key: %v", minersWallet.PrivateKeyStr())
		log.Printf("public key: %v", minersWallet.PublicKeyStr())
		log.Printf("blockchain address: %v", minersWallet.BlockchainAddress())
	}

	return bc
}

func (bcs *BlockchainServer) GetChain(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		bc := bcs.GetBlockchain()
		m, _ := bc.MarshalJSON()
		io.WriteString(w, string(m[:]))
	default:
		log.Printf("Error: Invalid HTTP Method: %v", req.Method)
	}
}

func (bcs *BlockchainServer) Transactions(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		w.Header().Add("Content-Type", "application/json")
		bc := bcs.GetBlockchain()
		transactions := bc.TransactionPool()
		m, _ := json.Marshal(struct {
			Transactions []*block.Transaction `json:"transactions"`
			Length       int                  `json:"length"`
		}{
			Transactions: transactions,
			Length:       len(transactions),
		})
		io.WriteString(w, string(m[:]))

	case http.MethodPost:
		decoder := json.NewDecoder(req.Body)
		var t block.TransactionRequest
		err := decoder.Decode(&t)

		if err != nil {
			log.Printf("Error %v", err)
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}

		if !t.Validate() {
			log.Println("ERROR: missing field(s)")
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}

		publicKey := utils.PublicKeyFromString(*t.SenderPublicKey)
		signature := utils.SignatureFromString(*t.Signature)

		bc := bcs.GetBlockchain()

		isCreated := bc.Createransaction(*t.SenderBlockchainAddress, *t.RecipientBlockchainAddress, block.Token{TokenName: *t.TokenName, TokenValue: *t.TokenValue}, publicKey, signature)

		w.Header().Add("Content-Type", "applications/json")
		var m []byte
		if !isCreated {
			w.WriteHeader(http.StatusBadRequest)
			m = utils.JsonStatus("failed")
		} else {
			w.WriteHeader(http.StatusCreated)
			m = utils.JsonStatus("success")
		}

		io.WriteString(w, string(m))

	case http.MethodPut:
		decoder := json.NewDecoder(req.Body)
		var t block.TransactionRequest
		err := decoder.Decode(&t)

		if err != nil {
			log.Printf("Error %v", err)
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}

		if !t.Validate() {
			log.Println("ERROR: missing field(s)")
			io.WriteString(w, string(utils.JsonStatus("fail")))
			return
		}

		publicKey := utils.PublicKeyFromString(*t.SenderPublicKey)
		signature := utils.SignatureFromString(*t.Signature)

		bc := bcs.GetBlockchain()

		isUpdated := bc.AddTransaction(*t.SenderBlockchainAddress, *t.RecipientBlockchainAddress, block.Token{TokenName: *t.TokenName, TokenValue: *t.TokenValue}, publicKey, signature)

		w.Header().Add("Content-Type", "applications/json")
		var m []byte
		if !isUpdated {
			w.WriteHeader(http.StatusBadRequest)
			m = utils.JsonStatus("failed")
		} else {
			m = utils.JsonStatus("success")
		}

		io.WriteString(w, string(m))

	case http.MethodDelete:
		bc := bcs.GetBlockchain()
		bc.ClearTransactionPool()
		io.WriteString(w, string(utils.JsonStatus("success")))
	default:
		log.Println("Error: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Mine(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		bc := bcs.GetBlockchain()
		isMined := bc.Mining()

		var m []byte

		if !isMined {
			w.WriteHeader(http.StatusBadRequest)
			m = utils.JsonStatus("failed")
		} else {
			m = utils.JsonStatus("success")
		}

		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m))
	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) StartMine(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		bc := bcs.GetBlockchain()
		bc.StartMining()

		m := utils.JsonStatus("success")

		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m))
	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) GetTokenBalance(w http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodGet:
		blockchainAddress := req.URL.Query().Get("blockchain_address")
		token := req.URL.Query().Get("token_name")

		//TODO:validation

		balance := bcs.GetBlockchain().CalculateTotalAmount(blockchainAddress, token)

		ar := &block.Token{TokenName: token, TokenValue: balance}

		m, _ := ar.MarshalJSON()

		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m[:]))

	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)

	}

}

func (bcs *BlockchainServer) GetTokenBalances(w http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodGet:
		blockchainAddress := req.URL.Query().Get("blockchain_address")
		//TODO:validation

		balances := bcs.GetBlockchain().CalculateAllAmounts(blockchainAddress)

		// ar := &block.Token{TokenName: token, TokenValue: balance}

		// m, _ := json.Marshal(balances)

		// w.Header().Add("Content-Type", "application/json")
		// io.WriteString(w, string(m[:]))

		ar := &block.AmountResponse{Amount: balances}
		m, _ := ar.MarshalJSON()

		w.Header().Add("Content-Type", "application/json")
		io.WriteString(w, string(m[:]))

	default:
		log.Println("ERROR: Invalid HTTP Method")
		w.WriteHeader(http.StatusBadRequest)

	}

}

func (bcs *BlockchainServer) Consensus(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPut:
		bc := bcs.GetBlockchain()
		replaced := bc.ResolveConflicts()

		w.Header().Add("Content-Type", "application/json")

		if replaced {
			io.WriteString(w, string(utils.JsonStatus("success")))
		} else {
			io.WriteString(w, string(utils.JsonStatus("failed")))
		}

	default:
		log.Printf("ERROR: unknown method %s", req.Method)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (bcs *BlockchainServer) Connect() {
	bcs.GetBlockchain().Run() //sync and start the nodes
	http.HandleFunc("/", bcs.GetChain)
	http.HandleFunc("/transactions", bcs.Transactions)
	http.HandleFunc("/mine", bcs.Mine)
	http.HandleFunc("/mine/start", bcs.StartMine)
	http.HandleFunc("/balance", bcs.GetTokenBalance)
	http.HandleFunc("/balance_all", bcs.GetTokenBalances)
	http.HandleFunc("/consensus", bcs.Consensus)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(int(bcs.port)), nil))
}
