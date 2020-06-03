package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"

	v "vault-secrets/vault_client"
)

type keepassEntry struct {
	group    string
	title    string
	username string
	password string
	url      string
	notes    string
}

type VaultEntries struct {
	entry interface{}
}

func main() {
	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken == "" {
		log.Fatal("VAULT_TOKEN env variable is empty")
	}
	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		log.Fatal("VAULT_ADDR env variable is empty")
	}

	c, err := v.NewClient(vaultToken)
	if err != nil {
		log.Fatalf("err: %s\n", err)
	}

	// TODO: add support being piped to.
	// gpg -d test-keepassdb.csv.gpg | go run main.go
	r, err := os.Open("test-keepassdb.csv")
	if err != nil {
		log.Fatalf("error: %s\n", err)
	}
	cr := csv.NewReader(r)
	records, err := cr.ReadAll()
	if err != nil {
		log.Fatalf("error: %s\n", err)
	}

	var entries []keepassEntry

	for _, e := range records {
		entry := keepassEntry{
			group:    e[0],
			title:    e[1],
			username: e[2],
			password: e[3],
			url:      e[4],
			notes:    e[5],
		}
		entries = append(entries, entry)
	}
	d := VaultEntries{
		entry: make(map[string]interface{}),
	}
	for i, e := range entries {
		if i == 0 {
			continue
		}
		secret, ok := d.entry.(map[string]interface{})
		if !ok {
			panic("d.entry is not a map")
		}
		//TODO: need to seal this secret in enclave?
		//https://github.com/awnumar/memguard/issues/118
		//https://github.com/genezhang/crypt
		secret["Group"] = e.group
		secret["Title"] = e.title
		secret["Username"] = e.username
		secret["Password"] = e.password
		secret["URL"] = e.url
		secret["Notes"] = e.notes
		secretPath := fmt.Sprintf("cubbyhole/%s", e.title)
		_, err := c.Client.Logical().Write(secretPath, secret)
		if err != nil {
			log.Fatalf("error while writing to secret path '%s' to Vault: %s\n", secretPath, err)
		}
		delete(secret, "Password")
	}
}
