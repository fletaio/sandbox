package main

import (
	"errors"
)

// account_tx errors
var (
	ErrInvalidSequence             = errors.New("invalid sequence")
	ErrInvalidTransactionSignature = errors.New("invalid transaction signature")
	ErrInvalidSignerCount          = errors.New("invalid signer count")
	ErrInvalidAccountSigner        = errors.New("invalid account signer")
	ErrInvalidAddress              = errors.New("invalid address")
	ErrInvalidPublicKey            = errors.New("invalid public key")
	ErrInvalidTxInCount            = errors.New("invalid txin count")
	ErrInvalidUTXO                 = errors.New("invalid utxo")
	ErrShortUserID                 = errors.New("short userid")
	ErrNotAllowed                  = errors.New("not allowed")
	ErrNotExistAccount             = errors.New("not exist account")
	ErrExistAddress                = errors.New("exist address")
	ErrExistKeyHash                = errors.New("exist key hash")
	ErrExistUserID                 = errors.New("exist userid")
	ErrExistAccountName            = errors.New("exist account name")
	ErrQueueFull                   = errors.New("queue full")
)
