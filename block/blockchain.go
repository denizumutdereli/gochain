package block

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"main/config"
	"main/utils"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/shopspring/decimal"
)

type Block struct {
	timestamp    int64
	nonce        int
	previousHash [32]byte
	transactions []*Transaction
}

type Blockchain struct {
	transactionPool   []*Transaction
	chain             []*Block
	blockchainAddress string
	port              uint16
	mux               sync.Mutex
	db                *badger.DB
	nodes             []string
	muxNodes          sync.Mutex
	conf              config.Blockchain
}

// func NewBlockchain(blockchainAddress string, port uint16) *Blockchain {
// 	b := &Block{}
// 	bc := new(Blockchain)
// 	bc.blockchainAddress = blockchainAddress
// 	bc.CreateBlock(0, b.Hash())
// 	bc.port = port
// 	return bc
// }

func CreateBlockchain(blockchainAddress string, conf config.Blockchain) (*Blockchain, error) {
	var lastHash [32]byte
	b := &Block{}
	bc := new(Blockchain)
	bc.blockchainAddress = blockchainAddress
	opts := badger.DefaultOptions(conf.DbSavePath)
	db, err := badger.Open(opts)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	err = db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {
			fmt.Println("no existing blockchain found")
			lastHash = b.Hash()
			fmt.Println("genesis created")
			err = txn.Set([]byte("lh"), lastHash[:])
			if err != nil {
				return fmt.Errorf("error occured while getting hash: %v", err)
			}
			return err
		} else {
			item, err := txn.Get([]byte("lh"))
			if err != nil {
				return fmt.Errorf("error occured while getting lasted hash: %v", err)
			}
			item.Value(func(val []byte) error {
				copy(lastHash[:], val)
				return nil
			})
			return err
		}
	})
	if err != nil {
		return nil, err
	}
	bc.db = db
	bc.port = conf.BlockChainPort
	fmt.Println(fmt.Sprintf("last hash: %x", lastHash))
	bc.CreateBlock(0, lastHash)
	return bc, nil
}

func NewBlock(nonce int, previousHash [32]byte, transactions []*Transaction) *Block {
	b := new(Block)
	b.timestamp = time.Now().UnixNano()
	b.nonce = nonce
	b.previousHash = previousHash
	b.transactions = transactions
	return b
}

func (b *Block) PreviousHash() [32]byte {
	return b.previousHash
}

func (b *Block) Nonce() int {
	return b.nonce
}

func (b *Block) Transactions() []*Transaction {
	return b.transactions
}

func (b *Block) Print() {
	fmt.Printf("timestamp       %d\n", b.timestamp)
	fmt.Printf("nonce           %d\n", b.nonce)
	fmt.Printf("previous_hash   %x\n", b.previousHash)
	for _, t := range b.transactions {
		t.Print()
	}
}

func (b *Block) Hash() [32]byte {
	m, _ := json.Marshal(b)
	return sha256.Sum256([]byte(m))
}

func (b *Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Timestamp int64 `json:"timestamp"`
		Nonce     int   `json:"nonce"`
		// PreviousHash [32]byte       `json:"previous_hash"`
		PreviousHash string         `json:"previous_hash"`
		Transactions []*Transaction `json:"transactions"`
	}{
		Timestamp:    b.timestamp,
		Nonce:        b.nonce,
		PreviousHash: fmt.Sprintf("%x", b.previousHash),
		Transactions: b.transactions,
	})
}

func (b *Block) UnmarshalJSON(data []byte) error {

	var previousHash string

	v := &struct {
		Timestamp    *int64          `json:"timestamp"`
		Nonce        *int            `json:"nonce"`
		PreviousHash *string         `json:"previous_hash"`
		Transactions *[]*Transaction `json:"transactions"`
	}{
		Timestamp:    &b.timestamp,
		Nonce:        &b.nonce,
		PreviousHash: &previousHash,
		Transactions: &b.transactions,
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	ph, _ := hex.DecodeString(*v.PreviousHash)
	copy(b.previousHash[:], ph[:32])

	return nil
}

func (bc *Blockchain) Chain() []*Block {
	return bc.chain
}

func (bc *Blockchain) Run() {
	bc.StartSyncNodes()
	bc.ResolveConflicts() //when connected it should be resolved
	bc.StartMining()
}

func (bc *Blockchain) SetNodes() {
	// bc.nodes = utils.FindP2P(
	// 	utils.GetHost(), bc.port,
	// 	NODE_IP_RANGE_START, NODE_IP_RANGE_END,
	// 	BLOCKCHAIN_PORT_RANGE_START, BLOCKCHAIN_PORT_RANGE_END)
	// log.Printf("%v", bc.nodes)
	bc.nodes = utils.FindP2P(
		"127.0.0.1", bc.port,
		bc.conf.IpRangeStart, bc.conf.IpRangeEnd,
		bc.conf.PortRangeStart, bc.conf.PortRangeEnd)
	log.Printf("%v", bc.nodes)
}

func (bc *Blockchain) SyncNodes() {
	bc.muxNodes.Lock()
	defer bc.muxNodes.Unlock()
	bc.SetNodes()
}

func (bc *Blockchain) StartSyncNodes() {
	bc.SyncNodes()
	_ = time.AfterFunc(time.Second*bc.conf.NodeSyncTimeSec, bc.StartSyncNodes)
}

func (bc *Blockchain) TransactionPool() []*Transaction {
	return bc.transactionPool
}

func (bc *Blockchain) ClearTransactionPool() {
	bc.transactionPool = bc.transactionPool[:0] //empty
}

func (bc *Blockchain) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Blocks []*Block `json:"chain"`
	}{
		Blocks: bc.chain,
	})
}

func (bc *Blockchain) UnmarshalJSON(data []byte) error {
	v := &struct {
		Blocks *[]*Block `json:"chain"`
	}{
		Blocks: &bc.chain,
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	return nil
}

func (bc *Blockchain) CreateBlock(nonce int, previousHash [32]byte) *Block {
	b := NewBlock(nonce, previousHash, bc.transactionPool)
	bc.chain = append(bc.chain, b)
	bc.transactionPool = []*Transaction{}
	bc.UpdateLastHash()

	for _, n := range bc.nodes {

		endpoint := fmt.Sprintf("http://%s/transactions", n)
		client := &http.Client{}

		req, err := http.NewRequest("DELETE", endpoint, nil)

		if err != nil {
			log.Printf("ERROR: %v", err)
		}

		resp, err := client.Do(req)

		if err != nil {
			log.Printf("ERROR: %v", err)
		}

		log.Printf("Response: %v", resp)

	}

	return b
}

func (bc *Blockchain) UpdateLastHash() bool {
	err := bc.db.Update(func(txn *badger.Txn) error {
		lastHash := bc.LastBlock().Hash()
		fmt.Println("last hash: ", fmt.Sprintf("%x", lastHash))
		err := txn.Set([]byte("lh"), lastHash[:])
		if err != nil {
			//TODO:rollback mechanism
			return err
		}
		item, _ := txn.Get([]byte("lh"))
		item.Value(func(val []byte) error {
			fmt.Println(fmt.Sprintf("last check %x", val))
			return nil
		})
		return err
	})
	return err == nil
}

func (bc *Blockchain) LastBlock() *Block {
	return bc.chain[len(bc.chain)-1]
}

func (bc *Blockchain) Print() {
	for i, block := range bc.chain {
		fmt.Printf("%s Chain %d %s\n", strings.Repeat("=", 25), i,
			strings.Repeat("=", 25))
		block.Print()
	}
	fmt.Printf("%s\n", strings.Repeat("*", 25))
}

func (bc *Blockchain) Createransaction(sender string, recipient string, token Token,
	senderPublicKey *ecdsa.PublicKey, s *utils.Signature) bool {
	isTransacted := bc.AddTransaction(sender, recipient, token, senderPublicKey, s)

	if isTransacted {
		for _, n := range bc.nodes {
			publicKeyStr := fmt.Sprintf("%064x%064x", senderPublicKey.X.Bytes(), senderPublicKey.Y.Bytes())
			signatureStr := s.String()
			bt := &TransactionRequest{&sender, &recipient, &publicKeyStr, &token.TokenName, &token.TokenValue, &signatureStr}
			m, err := json.Marshal(bt)

			if err != nil {
				log.Printf("ERROR: %v", err)
			}

			buf := bytes.NewBuffer(m)
			endpoint := fmt.Sprintf("http://%s/transactions", n)
			client := &http.Client{}

			req, err := http.NewRequest("PUT", endpoint, buf)

			if err != nil {
				log.Printf("ERROR: %v", err)
			}

			resp, err := client.Do(req)

			if err != nil {
				log.Printf("ERROR: %v", err)
			}

			log.Printf("Response: %v", resp)
		}
	}

	return isTransacted
}

func (bc *Blockchain) AddTransaction(sender string, recipient string, token Token,
	senderPublicKey *ecdsa.PublicKey, s *utils.Signature) bool {
	t := NewTransaction(sender, recipient, token)

	if sender == bc.conf.MiningSender {
		bc.transactionPool = append(bc.transactionPool, t)
		return true
	}

	if bc.VerifyTransactionSignature(senderPublicKey, s, t) {

		if !bc.CalculateTotalAmount(sender, token.TokenName).GreaterThan(token.TokenValue) {
			log.Println("ERROR: Not enough balance in a wallet")
			return false
		}

		bc.transactionPool = append(bc.transactionPool, t)
		return true
	} else {
		log.Println("ERROR: Verify Transaction")
	}
	return false

}

func (bc *Blockchain) VerifyTransactionSignature(
	senderPublicKey *ecdsa.PublicKey, s *utils.Signature, t *Transaction) bool {
	m, _ := json.Marshal(t)
	h := sha256.Sum256([]byte(m))
	return ecdsa.Verify(senderPublicKey, h[:], s.R, s.S)
}

func (bc *Blockchain) CopyTransactionPool() []*Transaction {
	transactions := make([]*Transaction, 0)
	for _, t := range bc.transactionPool {
		transactions = append(transactions, NewTransaction(t.senderBlockchainAddress, t.recipientBlockchainAddress, t.token))
	}
	return transactions
}

func (bc *Blockchain) ValidProof(nonce int, previousHash [32]byte, transactions []*Transaction, difficulty int) bool {
	zeros := strings.Repeat("0", difficulty)
	guessBlock := Block{0, nonce, previousHash, transactions}
	guessHashStr := fmt.Sprintf("%x", guessBlock.Hash())
	return guessHashStr[:difficulty] == zeros
}

func (bc *Blockchain) ProofOfWork() int {
	transactions := bc.CopyTransactionPool()
	previousHash := bc.LastBlock().Hash()
	nonce := 0
	for !bc.ValidProof(nonce, previousHash, transactions, bc.conf.Difficulty) {
		nonce += 1
	}
	return nonce
}

func (bc *Blockchain) Mining() bool {

	bc.mux.Lock()
	defer bc.mux.Unlock()

	// if len(bc.transactionPool) == 0 {
	// 	return false
	// }

	bc.AddTransaction(bc.conf.MiningSender, bc.blockchainAddress, Token{TokenName: bc.conf.DefaultRewardToken, TokenValue: utils.FloatToDecimal(bc.conf.MiningReward)}, nil, nil)
	nonce := bc.ProofOfWork()
	previousHash := bc.LastBlock().Hash()
	bc.CreateBlock(nonce, previousHash)
	log.Println("action=mining, status=success")

	for _, n := range bc.nodes {
		endpoint := fmt.Sprintf("http://%s/consensus", n)
		client := &http.Client{}
		req, err := http.NewRequest("PUT", endpoint, nil)

		if err != nil {
			fmt.Printf("Error on consensus request after mining: %v", err)
			return false
		}

		resp, err := client.Do(req)

		if err != nil {
			fmt.Printf("Error on consensus after mining: %v", err)
			return false
		}

		log.Printf("%v", resp)
	}
	return true
}

func (bc *Blockchain) StartMining() {
	bc.Mining()
	_ = time.AfterFunc(time.Second*bc.conf.MiningTimerSeconds, bc.StartMining) //lock logically and loop backwards
}

func (bc *Blockchain) CalculateTotalAmount(blockchainAddress string, tokenName string) decimal.Decimal {
	totalAmount := decimal.New(0.0, 0)

	for _, b := range bc.chain {
		for _, t := range b.transactions {
			if blockchainAddress == t.recipientBlockchainAddress {
				if tokenName == t.token.TokenName {
					totalAmount = totalAmount.Add(t.token.TokenValue)
				}
			}

			if blockchainAddress == t.senderBlockchainAddress {
				if tokenName == t.token.TokenName {
					totalAmount = totalAmount.Sub(t.token.TokenValue)
				}
			}

		}
	}

	return totalAmount
}

func (bc *Blockchain) CalculateAllAmounts(blockchainAddress string) []*Token {

	tokens := make(map[string]*Token, 0)

	for _, b := range bc.chain {
		for _, t := range b.transactions {
			totalAmount := bc.CalculateTotalAmount(blockchainAddress, t.token.TokenName)
			tokens[t.token.TokenName] = &Token{TokenName: t.token.TokenName, TokenValue: totalAmount}
		}
	}

	tokenSlice := make([]*Token, len(tokens))
	i := 0
	for _, v := range tokens {
		tokenSlice[i] = v
		i++
	}

	return tokenSlice
}

func (bc *Blockchain) ValidChain(chain []*Block) bool { //what if later on?
	preBlock := chain[0]
	currentIndex := 1
	for currentIndex < len(chain) { //TODO: last longest chain in given time
		b := chain[currentIndex]
		if b.previousHash != preBlock.Hash() {
			return false
		}

		if !bc.ValidProof(b.Nonce(), b.previousHash, b.transactions, bc.conf.Difficulty) {
			return false
		}

		preBlock = b
		currentIndex += 1
	}

	return true
}

func (bc *Blockchain) ResolveConflicts() bool {
	var longestChain []*Block = nil
	maxLength := len(bc.chain)

	for _, n := range bc.nodes {
		endpoint := fmt.Sprintf("http://%s/chain", n)
		resp, _ := http.Get(endpoint)
		if resp.StatusCode == 200 {
			var bcResp Blockchain
			decoder := json.NewDecoder(resp.Body)
			_ = decoder.Decode(&bcResp)

			chain := bcResp.Chain()

			if len(chain) > maxLength && bc.ValidChain(chain) {
				maxLength = len(chain)
				longestChain = chain
			}
		}
	}

	if longestChain != nil {
		bc.chain = longestChain
		log.Printf("Resovle confilicts replaced")
		return true
	}
	log.Printf("Resovle conflicts not replaced")
	return false
}

type AllTokensResponse struct {
	Results []*Token `json:"results"`
}

type Token struct {
	TokenName  string          `json:"token_name"`
	TokenValue decimal.Decimal `json:"token_value"`
}

func (t *Token) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		TokenName  string          `json:"token_name"`
		TokenValue decimal.Decimal `json:"token_value"`
	}{
		TokenName:  t.TokenName,
		TokenValue: t.TokenValue,
	})
}

type Transaction struct {
	senderBlockchainAddress    string
	recipientBlockchainAddress string
	token                      Token
}

func NewTransaction(sender string, recipient string, token Token) *Transaction {
	return &Transaction{sender, recipient, token}
}

func (tk *Token) Print() {
	fmt.Printf(tk.TokenName, "%.1f\n", tk.TokenValue)
}

func (t *Transaction) Print() {
	fmt.Printf("%s\n", strings.Repeat("-", 40))
	fmt.Printf(" sender_blockchain_address   = %s\n", t.senderBlockchainAddress)
	fmt.Printf(" recipient_blockchain_address   = %s\n", t.recipientBlockchainAddress)
	fmt.Printf(" token %s\n value = %s\n", t.token.TokenName, t.token.TokenValue)
	//fmt.Printf(" token value = %.1f\n", t.token)
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sender    string `json:"sender_blockchain_address"`
		Recipient string `json:"recipient_blockchain_address"`
		Token     Token  `json:"token"`
	}{
		Sender:    t.senderBlockchainAddress,
		Recipient: t.recipientBlockchainAddress,
		Token:     t.token,
	})
}

func (t *Transaction) UnmarshalJSON(data []byte) error {
	v := &struct {
		Sender    *string `json:"sender_blockchain_address"`
		Recipient *string `json:"recipient_blockchain_address"`
		Token     *Token  `json:"token"`
	}{
		Sender:    &t.senderBlockchainAddress,
		Recipient: &t.recipientBlockchainAddress,
		Token:     &t.token,
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	return nil
}

type TransactionRequest struct {
	SenderBlockchainAddress    *string          `json:"sender_blockchain_address"`
	RecipientBlockchainAddress *string          `json:"recipient_blockchain_address"`
	SenderPublicKey            *string          `json:"sender_public_key"`
	TokenName                  *string          `json:"token_name"`
	TokenValue                 *decimal.Decimal `json:"token_value"`
	Signature                  *string          `json:"signature"`
}

func (tr *TransactionRequest) Validate() bool {

	log.Println(tr)
	if tr.SenderBlockchainAddress == nil ||
		tr.RecipientBlockchainAddress == nil ||
		tr.SenderPublicKey == nil ||
		tr.TokenName == nil || tr.TokenValue == nil {
		return false
	}

	return true
}

type AmountResponse struct {
	Amount []*Token `json:"amount"`
}

func (ar *AmountResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Amount []*Token `json:"amount"`
	}{
		Amount: ar.Amount,
	})
}
