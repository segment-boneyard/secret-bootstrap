package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	Usage     = `./secret-bootstrap secretFile`
	VaultAddr = "http://vault.segment.local/v1"
	Metadata  = `http://169.254.169.254/latest/dynamic/instance-identity/pkcs7`
	IamInfo   = `http://169.254.169.254/latest/meta-data/iam/info`
	OutName   = "secrets.sources"
)

var (
	EnvLine = "%s=\"%s\""
)

type SecretFile map[string]string

type Outfile []string

type VaultAuthRequest struct {
	Role  string `json:"role"`
	PKCS7 string `json:"pkcs7"`
	Nonce string `json:"nonce"`
}

type VaultResponse struct {
	RequestID     string      `json:"request_id"`
	LeaseID       string      `json:"lease_id"`
	Renewable     bool        `json:"renewable"`
	LeaseDuration int         `json:"lease_duration"`
	Data          interface{} `json:"data"`
	WrapInfo      interface{} `json:"wrap_info"`
	Warnings      interface{} `json:"warnings"`
	Auth          struct {
		ClientToken string   `json:"client_token"`
		Accessor    string   `json:"accessor"`
		Policies    []string `json:"policies"`
		Metadata    struct {
			AccountID     string `json:"account_id"`
			AmiID         string `json:"ami_id"`
			InstanceID    string `json:"instance_id"`
			Nonce         string `json:"nonce"`
			Region        string `json:"region"`
			Role          string `json:"role"`
			RoleTagMaxTTL string `json:"role_tag_max_ttl"`
		} `json:"metadata"`
		LeaseDuration int  `json:"lease_duration"`
		Renewable     bool `json:"renewable"`
	} `json:"auth"`
}

type VaultSecretResponse struct {
	RequestID     string `json:"request_id"`
	LeaseID       string `json:"lease_id"`
	Renewable     bool   `json:"renewable"`
	LeaseDuration int    `json:"lease_duration"`
	Data          struct {
		Value string `json:"value"`
	} `json:"data"`
	WrapInfo interface{} `json:"wrap_info"`
	Warnings interface{} `json:"warnings"`
	Auth     interface{} `json:"auth"`
}

type InstanceProfile struct {
	Code               string
	LastUpdated        time.Time
	InstanceProfileArn string
	InstanceProfileId  string
}

func (o Outfile) String() string {
	var (
		leadingNewline = "\n%s"
		noNewline      = "%s"
	)
	s := ""
	for i, entry := range o {
		tmpl := leadingNewline
		if i == 0 {
			tmpl = noNewline
		}
		s = s + fmt.Sprintf(tmpl, entry)
	}
	return s
}

func printUsageAndExit() {
	log.Fatalf("%s\n", Usage)
}

func fetchIdentity() (string, error) {
	resp, err := http.Get(Metadata)
	if err != nil {
		return "", err
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	pkcs7 := string(buf)
	pkcs7 = strings.Replace(pkcs7, "\n", "", -1)
	return pkcs7, nil
}

func fetchRole() (string, error) {
	var (
		i InstanceProfile
	)
	resp, err := http.Get(IamInfo)
	if err != nil {
		return "", err
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(buf, &i)
	if err != nil {
		return "", err
	}
	roleArn := i.InstanceProfileArn
	arnParts := strings.Split(roleArn, "/")
	if len(arnParts) != 2 {
		return "", errors.New("Could not parse instance profile arn")
	}
	role := arnParts[1]
	return role, nil
}

func AuthVault() (string, error) {
	pkcs7, err := fetchIdentity()
	if err != nil {
		return "", err
	}
	role, err := fetchRole()
	if err != nil {
		return "", err
	}

	ar := &VaultAuthRequest{
		Role:  role,
		Nonce: "AAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		PKCS7: pkcs7,
	}
	body, err := json.Marshal(ar)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/aws/login", VaultAddr), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	v := new(VaultResponse)
	client := &http.Client{}
	resp, err := client.Do(req)
	authResponse, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(authResponse, v)
	if err != nil {
		return "", err
	}
	return v.Auth.ClientToken, nil
}

func fetchSecret(name, tok string) (string, error) {
	var (
		client = &http.Client{}
		v      VaultSecretResponse
	)
	req, err := http.NewRequest("GET", fmt.Sprintf("http://vault.segment.local/v1/secret/%s", name), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("X-Vault-Token", tok)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(buf, &v)
	if err != nil {
		return "", err
	}
	if len(v.Data.Value) == 0 {
		return "", errors.New("Could not fetch secret. No secret found in vault response")
	}
	return v.Data.Value, nil
}

func FetchSecrets(s SecretFile) (Outfile, error) {
	var (
		o Outfile
	)
	tok, err := AuthVault()
	if err != nil {
		log.Fatalf("Could not fetch vault auth token.")
	}

	for envName, vaultName := range s {
		secret, err := fetchSecret(vaultName, tok)
		// If we are missing a single secret it is fatal. An application may start
		// normally and have unexpected behavior if it is missing one of its secrets.
		if err != nil {
			log.Fatalf("Could not fetch secret %s: %s", vaultName, err)
		}
		s := fmt.Sprintf(EnvLine, envName, secret)
		o = append(o, s)
	}
	return o, nil
}

func main() {
	var (
		s SecretFile
	)
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		printUsageAndExit()
	}

	buf, err := ioutil.ReadFile(args[0])
	if err != nil {
		log.Fatalf("Could not read secret file %s: %s", args[0], err)
	}

	err = json.Unmarshal(buf, &s)
	if err != nil {
		log.Fatalf("Could not parse secret file %s: %s", args[0], err)
	}
	outfile, err := FetchSecrets(s)
	ioutil.WriteFile(OutName, []byte(outfile.String()), 0444)
}
