package main

import (
	"bytes"
	"encoding/json"

	"github.com/keybase/go-keychain"
)

type darwinStorage struct {
}

func CreateStorage() Storage {
	return &darwinStorage{}
}

func (s *darwinStorage) Get() (Config, error) {
	q := configItem()
	q.SetMatchLimit(keychain.MatchLimitOne)
	q.SetReturnData(true)
	results, err := keychain.QueryItem(q)
	if err != nil {
		return Config{}, err
	} else if len(results) != 1 {
		return Config{}, nil
	}
	var config Config
	err = json.NewDecoder(bytes.NewReader(results[0].Data)).Decode(&config)
	return config, err
}

func (s *darwinStorage) Store(config Config) error {
	var data bytes.Buffer
	if err := json.NewEncoder(&data).Encode(config); err != nil {
		return err
	}
	q := configItem()
	item := configItem()
	item.SetData(data.Bytes())
	if err := keychain.UpdateItem(q, item); err == keychain.ErrorItemNotFound {
		return keychain.AddItem(item)
	} else {
		return err
	}
}

func configItem() keychain.Item {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService("pcloud")
	item.SetAccount("pcloud")
	item.SetLabel("pcloud-config")
	item.SetAccessGroup("me.lekva.pcloud")
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)
	return item
}
