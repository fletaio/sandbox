package main

import (
	"github.com/fletaio/common"
	"github.com/fletaio/core/data"
)

type mockRewarder struct {
}

// ApplyGenesis init genesis data
func (rd *mockRewarder) ApplyGenesis(ctx *data.ContextData) ([]byte, error) {
	SaveData, err := rd.buildSaveData()
	if err != nil {
		return nil, err
	}
	return SaveData, nil
}

// ProcessReward gives a reward to the block generator address
func (rd *mockRewarder) ProcessReward(addr common.Address, ctx *data.Context) ([]byte, error) {
	SaveData, err := rd.buildSaveData()
	if err != nil {
		return nil, err
	}
	return SaveData, nil
}

func (rd *mockRewarder) buildSaveData() ([]byte, error) {
	return []byte("test"), nil
}

// LoadFromSaveData recover the status using the save data
func (rd *mockRewarder) LoadFromSaveData(SaveData []byte) error {
	return nil
}
