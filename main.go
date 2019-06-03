package main

import (
	"bytes"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fletaio/common"
	"github.com/fletaio/common/hash"
	"github.com/fletaio/common/util"
	"github.com/fletaio/core/block"
	"github.com/fletaio/core/data"
	"github.com/fletaio/core/formulator"
	"github.com/fletaio/core/kernel"
	"github.com/fletaio/core/key"
	"github.com/fletaio/core/message_def"
	"github.com/fletaio/core/observer"
	"github.com/fletaio/core/transaction"
	"github.com/fletaio/framework/peer"
	"github.com/fletaio/framework/router"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
)

// consts
const (
	CreateAccountChannelSize = 100
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	var Key key.Key
	if bs, err := hex.DecodeString("b3055b3109f80e2a96a520f042fc7552931b3fa4e35079253e6ed69bce72c1bc"); err != nil {
		panic(err)
	} else {
		k, err := key.NewMemoryKeyFromBytes(bs)
		if err != nil {
			panic(err)
		}
		Key = k
	}

	obstrs := []string{
		"cd7cca6359869f4f58bb31aa11c2c4825d4621406f7b514058bc4dbe788c29be",
		"d8744df1e76a7b76f276656c48b68f1d40804f86518524d664b676674fccdd8a",
		"387b430fab25c03313a7e987385c81f4b027199304e2381561c9707847ec932d",
		"a99fa08114f41eb7e0a261cf11efdc60887c1d113ea6602aaf19eca5c3f5c720",
		"a9878ff3837700079fbf187c86ad22f1c123543a96cd11c53b70fedc3813c27b",
	}
	obkeys := make([]key.Key, 0, len(obstrs))
	ObserverKeys := make([]common.PublicHash, 0, len(obstrs))

	NetAddressMap := map[common.PublicHash]string{}
	NetAddressMapForFr := map[common.PublicHash]string{}
	for i, v := range obstrs {
		if bs, err := hex.DecodeString(v); err != nil {
			panic(err)
		} else if Key, err := key.NewMemoryKeyFromBytes(bs); err != nil {
			panic(err)
		} else {
			obkeys = append(obkeys, Key)
			Num := strconv.Itoa(i + 1)
			pubhash := common.NewPublicHash(Key.PublicKey())
			NetAddressMap[pubhash] = "127.0.0.1:300" + Num
			NetAddressMapForFr[pubhash] = "127.0.0.1:500" + Num
			ObserverKeys = append(ObserverKeys, pubhash)
		}
	}
	ObserverKeyMap := map[common.PublicHash]bool{}
	for _, pubhash := range ObserverKeys {
		ObserverKeyMap[pubhash] = true
	}

	frstrs := []string{
		"67066852dd6586fa8b473452a66c43f3ce17bd4ec409f1fff036a617bb38f063",
	}

	frkeys := make([]key.Key, 0, len(frstrs))
	for _, v := range frstrs {
		if bs, err := hex.DecodeString(v); err != nil {
			panic(err)
		} else if Key, err := key.NewMemoryKeyFromBytes(bs); err != nil {
			panic(err)
		} else {
			frkeys = append(frkeys, Key)
		}
	}

	ObserverHeights := []uint32{}

	obs := []*observer.Observer{}
	for _, obkey := range obkeys {
		GenCoord := common.NewCoordinate(0, 0)
		act := data.NewAccounter(GenCoord)
		tran := data.NewTransactor(GenCoord)
		evt := data.NewEventer(GenCoord)
		if err := initChainComponent(act, tran, evt); err != nil {
			panic(err)
		}
		if err := initSandboxComponent(act, tran, evt); err != nil {
			panic(err)
		}

		GenesisContextData, err := initGenesisContextData(act, tran, evt)
		if err != nil {
			panic(err)
		}

		StoreRoot := "./observer/" + common.NewPublicHash(obkey.PublicKey()).String()

		ks, err := kernel.NewStore(StoreRoot+"/kernel", 1, act, tran, evt, true)
		if err != nil {
			panic(err)
		}

		rd := &mockRewarder{}
		kn, err := kernel.NewKernel(&kernel.Config{
			ChainCoord:              GenCoord,
			ObserverKeyMap:          ObserverKeyMap,
			MaxBlocksPerFormulator:  8,
			MaxTransactionsPerBlock: 5000,
		}, ks, rd, GenesisContextData)
		if err != nil {
			panic(err)
		}

		cfg := &observer.Config{
			ChainCoord:     GenCoord,
			Key:            obkey,
			ObserverKeyMap: NetAddressMap,
		}
		ob, err := observer.NewObserver(cfg, kn)
		if err != nil {
			panic(err)
		}
		obs = append(obs, ob)

		ObserverHeights = append(ObserverHeights, kn.Provider().Height())
	}

	Formulators := []string{}
	FormulatorHeights := []uint32{}

	var GameKernel *kernel.Kernel
	frs := []*formulator.Formulator{}
	for _, frkey := range frkeys {
		GenCoord := common.NewCoordinate(0, 0)
		act := data.NewAccounter(GenCoord)
		tran := data.NewTransactor(GenCoord)
		evt := data.NewEventer(GenCoord)
		if err := initChainComponent(act, tran, evt); err != nil {
			panic(err)
		}
		if err := initSandboxComponent(act, tran, evt); err != nil {
			panic(err)
		}

		GenesisContextData, err := initGenesisContextData(act, tran, evt)
		if err != nil {
			panic(err)
		}

		StoreRoot := "./formulator/" + common.NewPublicHash(frkey.PublicKey()).String()

		//os.RemoveAll(StoreRoot)

		ks, err := kernel.NewStore(StoreRoot+"/kernel", 1, act, tran, evt, true)
		if err != nil {
			panic(err)
		}

		rd := &mockRewarder{}
		kn, err := kernel.NewKernel(&kernel.Config{
			ChainCoord:              GenCoord,
			ObserverKeyMap:          ObserverKeyMap,
			MaxBlocksPerFormulator:  8,
			MaxTransactionsPerBlock: 5000,
		}, ks, rd, GenesisContextData)
		if err != nil {
			panic(err)
		}

		cfg := &formulator.Config{
			Key:            frkey,
			ObserverKeyMap: NetAddressMapForFr,
			Formulator:     common.MustParseAddress("3CUsUpvEK"),
			Router: router.Config{
				Network: "tcp",
				Port:    7000,
			},
			Peer: peer.Config{
				StorePath: StoreRoot + "/peers",
			},
		}
		fr, err := formulator.NewFormulator(cfg, kn)
		if err != nil {
			panic(err)
		}
		frs = append(frs, fr)

		Formulators = append(Formulators, cfg.Formulator.String())
		FormulatorHeights = append(FormulatorHeights, kn.Provider().Height())

		GameKernel = kn
	}

	for i, ob := range obs {
		go func(BindOb string, BindFr string, ob *observer.Observer) {
			ob.Run(BindOb, BindFr)
		}(":300"+strconv.Itoa(i+1), ":500"+strconv.Itoa(i+1), ob)
	}

	for _, fr := range frs {
		go func(fr *formulator.Formulator) {
			fr.Run()
		}(fr)
	}

	var CreateAccountChannelID uint64
	var UsingChannelCount int64

	ew := NewEventWatcher()
	GameKernel.AddEventHandler(ew)

	e := echo.New()
	web := NewWebServer(e, "./webfiles")
	e.Renderer = web
	web.SetupStatic(e, "/public", "./webfiles/public")
	webChecker := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			web.CheckWatch()
			return next(c)
		}
	}
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
		}
		c.Logger().Error(err)
		c.HTML(code, err.Error())
	}

	e.GET("/websocket/:address", func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
		if err != nil {
			return err
		}
		defer conn.Close()

		bs := make([]byte, 32)
		crand.Read(bs)
		h := hash.Hash(bs)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(h.String())); err != nil {
			return err
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		sig, err := common.ParseSignature(string(msg))
		if err != nil {
			return err
		}

		var pubhash common.PublicHash
		if pubkey, err := common.RecoverPubkey(h, sig); err != nil {
			return err
		} else {
			pubhash = common.NewPublicHash(pubkey)
		}

		addrStr := c.Param("address")
		addr, err := common.ParseAddress(addrStr)
		if err != nil {
			return err
		}
		a, err := GameKernel.Loader().Account(addr)
		if err != nil {
			return err
		}
		acc := a.(*Account)
		if !acc.KeyHash.Equal(pubhash) {
			return ErrInvalidAddress
		}

		ew.AddWriter(addr, conn)
		defer ew.RemoveWriter(addr, conn)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return nil
			}
		}
	})

	e.GET("/", func(c echo.Context) error {
		args := make(map[string]interface{})
		args["test_arg_key"] = "test_arg_value"
		return c.Render(http.StatusOK, "index.html", args)
	}, webChecker)

	gAPI := e.Group("/api")
	gAPI.GET("/chain/height", func(c echo.Context) error {
		res := &WebHeightRes{
			Height: int(GameKernel.Provider().Height()),
		}
		return c.JSON(http.StatusOK, res)
	})
	gAPI.GET("/accounts", func(c echo.Context) error {
		pubkeyStr := c.QueryParam("pubkey")
		if len(pubkeyStr) == 0 {
			return ErrInvalidPublicKey
		}

		pubkey, err := common.ParsePublicKey(pubkeyStr)
		if err != nil {
			return err
		}

		pubhash := common.NewPublicHash(pubkey)
		KeyHashID := append(PrefixKeyHash, pubhash[:]...)

		loader := GameKernel.Loader()
		var rootAddress common.Address
		if bs := loader.AccountData(rootAddress, KeyHashID); len(bs) > 0 {
			var addr common.Address
			copy(addr[:], bs)
			utxos := []uint64{}
			for i := 0; i < GameCommandChannelSize; i++ {
				utxos = append(utxos, util.BytesToUint64(loader.AccountData(addr, []byte("utxo"+strconv.Itoa(i)))))
			}
			res := &WebAccountRes{
				Address: addr.String(),
				UTXOs:   utxos,
			}
			return c.JSON(http.StatusOK, res)
		} else {
			return ErrNotExistAccount
		}
	})
	gAPI.POST("/accounts", func(c echo.Context) error {
		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		defer c.Request().Body.Close()

		var req WebAccountReq
		if err := json.Unmarshal(body, &req); err != nil {
			return err
		}

		if len(req.UserID) < 4 {
			return ErrShortUserID
		}

		pubkey, err := common.ParsePublicKey(req.PublicKey)
		if err != nil {
			return err
		}

		pubhash := common.NewPublicHash(pubkey)
		KeyHashID := append(PrefixKeyHash, pubhash[:]...)
		UserIDHashID := append(PrefixUserID, []byte(req.UserID)...)

		var TxHash hash.Hash256
		loader := GameKernel.Loader()
		var rootAddress common.Address
		if bs := loader.AccountData(rootAddress, KeyHashID); len(bs) > 0 {
			var addr common.Address
			copy(addr[:], bs)
			utxos := []uint64{}
			for i := 0; i < GameCommandChannelSize; i++ {
				utxos = append(utxos, util.BytesToUint64(loader.AccountData(addr, []byte("utxo"+strconv.Itoa(i)))))
			}
			res := &WebAccountRes{
				Address: addr.String(),
				UTXOs:   utxos,
			}
			return c.JSON(http.StatusOK, res)
		}
		if bs := loader.AccountData(rootAddress, UserIDHashID); len(bs) > 0 {
			return ErrExistUserID
		}

		t, err := loader.Transactor().NewByType(CreateAccountTransctionType)
		if err != nil {
			return err
		}

		defer atomic.AddInt64(&UsingChannelCount, -1)
		if atomic.AddInt64(&UsingChannelCount, 1) >= CreateAccountChannelSize {
			return ErrQueueFull
		}

		cnid := atomic.AddUint64(&CreateAccountChannelID, 1) % CreateAccountChannelSize

		utxoid := util.BytesToUint64(loader.AccountData(rootAddress, []byte("utxo"+strconv.FormatUint(cnid, 10))))

		tx := t.(*CreateAccountTx)
		tx.Timestamp_ = uint64(time.Now().UnixNano())
		tx.Vin = []*transaction.TxIn{transaction.NewTxIn(utxoid)}
		tx.KeyHash = pubhash
		tx.UserID = req.UserID

		TxHash = tx.Hash()

		if sig, err := Key.Sign(TxHash); err != nil {
			return err
		} else if err := GameKernel.AddTransaction(tx, []common.Signature{sig}); err != nil {
			return err
		}

		timer := time.NewTimer(10 * time.Second)

		cp := GameKernel.Provider()
		SentHeight := cp.Height()
		for {
			select {
			case <-timer.C:
				return c.NoContent(http.StatusRequestTimeout)
			default:
				height := cp.Height()
				if SentHeight < height {
					SentHeight = height

					var rootAddress common.Address
					if bs := loader.AccountData(rootAddress, KeyHashID); len(bs) > 0 {
						var addr common.Address
						copy(addr[:], bs)
						utxos := []uint64{}
						for i := 0; i < GameCommandChannelSize; i++ {
							utxos = append(utxos, util.BytesToUint64(loader.AccountData(addr, []byte("utxo"+strconv.Itoa(i)))))
						}
						res := &WebAccountRes{
							Address: addr.String(),
							UTXOs:   utxos,
						}
						return c.JSON(http.StatusOK, res)
					}
				}
				time.Sleep(250 * time.Millisecond)
			}
		}
	})
	gAPI.GET("/games/:address", func(c echo.Context) error {
		addrStr := c.Param("address")
		addr, err := common.ParseAddress(addrStr)
		if err != nil {
			return err
		}

		var bs []byte
		var Height uint32
		var loader data.Loader
		if e := ew.LastBlockEvent(); e != nil {
			bs = e.ctx.AccountData(addr, []byte("game"))
			Height = e.b.Header.Height()
			loader = e.ctx
		} else {
			loader = GameKernel.Loader()
			GameKernel.Lock()
			bs = loader.AccountData(addr, []byte("game"))
			Height = GameKernel.Provider().Height()
			GameKernel.Unlock()
		}

		if len(bs) == 0 {
			return ErrNotExistAccount
		}

		gd := NewGameData()
		if _, err := gd.ReadFrom(bytes.NewReader(bs)); err != nil {
			return err
		}

		//////////////////////////////////////////////////////////////////////
		// Sandbox Area Begin
		//////////////////////////////////////////////////////////////////////

		res := &WebGameRes{
			Height: int(Height),
			Count:  int(gd.Count),
		}

		//////////////////////////////////////////////////////////////////////
		// Sandbox Area End
		//////////////////////////////////////////////////////////////////////

		return c.JSON(http.StatusOK, res)
	})
	gAPI.POST("/games/:address/commands/add_count", func(c echo.Context) error {
		addrStr := c.Param("address")
		addr, err := common.ParseAddress(addrStr)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		defer c.Request().Body.Close()

		var req WebAddCountReq
		if err := json.Unmarshal(body, &req); err != nil {
			return err
		}
		if req.UTXO == 0 {
			return ErrInvalidUTXO
		}

		//////////////////////////////////////////////////////////////////////
		// Sandbox Area Begin
		//////////////////////////////////////////////////////////////////////

		if req.Count == 0 {
			return ErrInvalidCount
		}

		//////////////////////////////////////////////////////////////////////
		// Sandbox Area End
		//////////////////////////////////////////////////////////////////////

		loader := GameKernel.Loader()
		if is, err := loader.IsExistAccount(addr); err != nil {
			return err
		} else if !is {
			return ErrNotExistAccount
		}

		//////////////////////////////////////////////////////////////////////
		// Sandbox Area Begin
		//////////////////////////////////////////////////////////////////////

		t, err := loader.Transactor().NewByType(AddCountTransactionType)
		if err != nil {
			return err
		}

		tx := t.(*AddCountTx)
		tx.Timestamp_ = uint64(time.Now().UnixNano())
		tx.Vin = []*transaction.TxIn{transaction.NewTxIn(req.UTXO)}
		tx.Address = addr
		tx.Count = uint64(req.Count)

		//////////////////////////////////////////////////////////////////////
		// Sandbox Area End
		//////////////////////////////////////////////////////////////////////

		var buffer bytes.Buffer
		if _, err := tx.WriteTo(&buffer); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, &WebTxRes{
			Type:    int(tx.Type()),
			TxHex:   hex.EncodeToString(buffer.Bytes()),
			HashHex: tx.Hash().String(),
		})
	})
	gAPI.POST("/games/:address/commands/commit", func(c echo.Context) error {
		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		defer c.Request().Body.Close()

		var req WebCommitReq
		if err := json.Unmarshal(body, &req); err != nil {
			return err
		}

		txBytes, err := hex.DecodeString(req.TxHex)
		if err != nil {
			return err
		}

		sigBytes, err := hex.DecodeString(req.SigHex)
		if err != nil {
			return err
		}

		loader := GameKernel.Loader()

		tx, err := loader.Transactor().NewByType(transaction.Type(req.Type))
		if err != nil {
			return err
		}
		if _, err := tx.ReadFrom(bytes.NewReader(txBytes)); err != nil {
			return err
		}

		var sig common.Signature
		if _, err := sig.ReadFrom(bytes.NewReader(sigBytes)); err != nil {
			return err
		}

		if err := GameKernel.AddTransaction(tx, []common.Signature{sig}); err != nil {
			return err
		}
		return c.NoContent(http.StatusOK)
	})

	go ew.Run()
	e.Start(":8080")
}

type BlockEvent struct {
	b   *block.Block
	ctx *data.Context
}

// EventWatcher TODO
type EventWatcher struct {
	sync.Mutex
	writerMap      map[common.Address]*websocket.Conn
	blockEventChan chan *BlockEvent
	lastBlockEvent *BlockEvent
}

// NewEventWatcher returns a EventWatcher
func NewEventWatcher() *EventWatcher {
	ew := &EventWatcher{
		writerMap:      map[common.Address]*websocket.Conn{},
		blockEventChan: make(chan *BlockEvent, 10000),
	}
	return ew
}

// LastBlockEvent returns the last block event
func (ew *EventWatcher) LastBlockEvent() *BlockEvent {
	ew.Lock()
	defer ew.Unlock()
	return ew.lastBlockEvent
}

// AddWriter TODO
func (ew *EventWatcher) AddWriter(addr common.Address, w *websocket.Conn) {
	ew.Lock()
	defer ew.Unlock()

	if old, has := ew.writerMap[addr]; has {
		old.WriteJSON(&WebNotify{
			Type: "__CONNECT__",
		})
		old.Close()
	}
	ew.writerMap[addr] = w
}

// RemoveWriter TODO
func (ew *EventWatcher) RemoveWriter(addr common.Address, w *websocket.Conn) {
	ew.Lock()
	defer ew.Unlock()

	if old, has := ew.writerMap[addr]; has {
		old.Close()
	}
	delete(ew.writerMap, addr)
}

// Run TODO
func (ew *EventWatcher) Run() {
	for {
		select {
		case e := <-ew.blockEventChan:
			ew.processBlock(e.b, e.ctx)
		}
	}
}

func (ew *EventWatcher) processBlock(b *block.Block, ctx *data.Context) {
	for i, t := range b.Body.Transactions {
		switch tx := t.(type) {
		case *AddCountTx:
			noti, err := getWebNotify(ctx, tx.Address, b.Header.Height(), uint16(i))
			if err != nil {
				continue
			}
			noti.Type = "add_count"
			noti.Count = int(tx.Count)
			ew.Notify(tx.Address, noti)
		}
	}
}

func getWebNotify(ctx *data.Context, addr common.Address, height uint32, index uint16) (*WebNotify, error) {
	id := transaction.MarshalID(height, index, 0)
	var errorMsg string
	for i := 0; i < GameCommandChannelSize; i++ {
		bs := ctx.AccountData(addr, []byte("utxo"+strconv.Itoa(i)))
		if len(bs) < 8 {
			continue
		}
		newid := util.BytesToUint64(bs)
		if id == newid {
			bs := ctx.AccountData(addr, []byte("result"+strconv.Itoa(i)))
			if len(bs) > 0 {
				errorMsg = string(bs)
			}
			break
		}
	}
	return &WebNotify{
		Height: int(height),
		UTXO:   int(id),
		Error:  errorMsg,
	}, nil
}

// OnProcessBlock called when processing a block to the chain (error prevent processing block)
func (ew *EventWatcher) OnProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	return nil
}

// OnPushTransaction called when pushing a transaction to the transaction pool (error prevent push transaction)
func (ew *EventWatcher) OnPushTransaction(kn *kernel.Kernel, tx transaction.Transaction, sigs []common.Signature) error {
	return nil
}

// AfterProcessBlock called when processed block to the chain
func (ew *EventWatcher) AfterProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) {
	e := &BlockEvent{
		b:   b,
		ctx: ctx,
	}

	ew.Lock()
	ew.lastBlockEvent = e
	ew.Unlock()

	ew.blockEventChan <- e
}

func (ew *EventWatcher) AfterPushTransaction(kn *kernel.Kernel, tx transaction.Transaction, sigs []common.Signature) {
}

// DoTransactionBroadcast called when a transaction need to be broadcast
func (ew *EventWatcher) DoTransactionBroadcast(kn *kernel.Kernel, msg *message_def.TransactionMessage) {
}

// DebugLog TEMP
func (ew *EventWatcher) DebugLog(kn *kernel.Kernel, args ...interface{}) {}

// Notify TODO
func (ew *EventWatcher) Notify(addr common.Address, noti *WebNotify) {
	ew.Lock()
	defer ew.Unlock()

	if conn, has := ew.writerMap[addr]; has {
		conn.WriteJSON(noti)
	}
}

type WebAccountReq struct {
	PublicKey string `json:"public_key"`
	UserID    string `json:"user_id"`
	Reward    string `json:"reward"`
}

type WebAccountRes struct {
	Address string   `json:"address"`
	UTXOs   []uint64 `json:"utxos"`
}

type WebHeightRes struct {
	Height int `json:"height"`
}

type WebTxRes struct {
	Type    int    `json:"type"`
	TxHex   string `json:"tx_hex"`
	HashHex string `json:"hash_hex"`
}

type WebCommitReq struct {
	Type   int    `json:"type"`
	TxHex  string `json:"tx_hex"`
	SigHex string `json:"sig_hex"`
}
