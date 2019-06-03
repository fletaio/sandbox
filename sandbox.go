package main

import (
	"errors"
	"io"
	"log"

	"github.com/fletaio/common/util"
	"github.com/fletaio/core/amount"
	"github.com/fletaio/core/data"
	"github.com/fletaio/core/transaction"
)

//////////////////////////////////////////////////////////////////////
// Sandbox Area Begin
//////////////////////////////////////////////////////////////////////

// consts
const (
	GameCommandChannelSize = 10
)

// errors
var (
	ErrInvalidCount = errors.New("invalid count")
)

type WebAddCountReq struct {
	UTXO  uint64 `json:"utxo"` // DO NOT CHANGE
	Count int    `json:"count"`
}

type WebNotify struct {
	Height int    `json:"height"` // DO NOT CHANGE
	Type   string `json:"type"`   // DO NOT CHANGE
	Count  int    `json:"count"`
	UTXO   int    `json:"utxo"`  // DO NOT CHANGE
	Error  string `json:"error"` // DO NOT CHANGE
}

type WebGameRes struct {
	Height int `json:"height"` // DO NOT CHANGE
	Count  int `json:"count"`
}

// transaction_type transaction types
const (
	// You can define [11 - 59] Transaction type
	// In sandbox, Transaction fee is not calculated
	// Game Transactions
	AddCountTransactionType = transaction.Type(11)
)

func initSandboxComponent(act *data.Accounter, tran *data.Transactor, evt *data.Eventer) error {
	TxFeeTable := map[string]*txFee{
		"sandbox.AddCount": &txFee{AddCountTransactionType, amount.COIN.MulC(10)},
		// ADD YOUR OWN TRANSACTION TO HERE
	}
	for name, item := range TxFeeTable {
		if err := tran.RegisterType(name, item.Type, item.Fee); err != nil {
			log.Println(name, item, err)
			return err
		}
	}
	return nil
}

// GameData stores all data of the game
type GameData struct {
	Count uint64
}

// NewGameData returns a GameData
func NewGameData() *GameData {
	gd := &GameData{
		Count: 0,
	}
	return gd
}

// WriteTo is a serialization function
func (gd *GameData) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint64(w, gd.Count); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (gd *GameData) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		gd.Count = v
	}
	return read, nil
}

//////////////////////////////////////////////////////////////////////
// Sandbox Area End
//////////////////////////////////////////////////////////////////////
