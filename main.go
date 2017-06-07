package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/segmentio/events"
	_ "github.com/segmentio/events/ecslogs"
	_ "github.com/segmentio/events/text"
	"github.com/segmentio/objconv/json"
)

var (
	vaultAddr       string
	ec2MetadataAddr string
)

func init() {
	vaultAddr = getenv("SECRET_BOOTSTRAP_AUTH_VAULT_ADDR", "vault.segment.local")
	ec2MetadataAddr = getenv("SECRET_BOOTSTRAP_EC2_METADATA_ADDR", "169.254.169.254")
}

func main() {
	argv := os.Args[1:]
	if len(argv) == 0 {
		usage("missing IAM role name")
	}

	role, vars, args := splitRoleVarsArgs(argv)
	if len(args) == 0 {
		usage("missing command to run after '--'")
	}

	path, err := exec.LookPath(args[0])
	if err != nil {
		fatal("%s: %s", args[0], err)
	}

	token, err := authVault()
	if err != nil {
		fatal("%s", err)
	}

	wg := sync.WaitGroup{}

	for _, name := range vars {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			value, err := fetchSecret(token, role, name)
			if err != nil {
				events.Log("error fetching secret value of %{variable}s: %{error}s", name, err)
			}
			// On error this will set the environment variable to an empty value,
			// otherwise the program won't be able to tell if it should use a
			// default value it may have, or if something went wrong.
			os.Setenv(name, value)
		}(name)
	}

	wg.Wait()
	syscall.Exec(path, args, os.Environ())
}

func splitRoleVarsArgs(argv []string) (role string, vars []string, args []string) {
	role, vars = argv[0], argv[1:]

	for i, v := range vars {
		if v == "--" {
			vars, args = vars[:i], vars[i+1:]
			break
		}
	}

	return
}

func authVault() (string, error) {
	events.Log("authenticating against vault")

	pkcs7, err := fetchIdentity()
	if err != nil {
		return "", err
	}

	role, err := fetchRole()
	if err != nil {
		return "", err
	}

	var authReq = struct {
		Role  string `json:"role"`
		PKCS7 string `json:"pkcs7"`
		Nonce string `json:"nonce"`
	}{
		Role:  role,
		PKCS7: pkcs7,
		Nonce: "AAAAAAAAAAAAAAAAAAAAAAAAAAAA", // should we change this?
	}

	body, _ := json.Marshal(authReq)
	req, _ := http.NewRequest("POST", "http://"+vaultAddr+"/auth/aws/login", bytes.NewReader(body))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var authRes struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}

	if err := json.NewDecoder(res.Body).Decode(&authRes); err != nil {
		return "", fmt.Errorf("failed to parse the vault authentication response: %s", err)
	}

	return authRes.Auth.ClientToken, nil
}

func fetchIdentity() (string, error) {
	events.Log("fetching EC2 instance PKCS7 identity from %{address}s", ec2MetadataAddr)

	res, err := http.Get("http://" + ec2MetadataAddr + "/latest/dynamic/instance-identity/pkcs7")
	if err != nil {
		return "", err
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	pkcs7 := string(buf)
	pkcs7 = strings.Replace(pkcs7, "\n", "", -1)
	return pkcs7, nil
}

func fetchRole() (string, error) {
	events.Log("fetching EC2 instance role from %{address}s", ec2MetadataAddr)

	var profile struct {
		Code               string
		LastUpdated        time.Time
		InstanceProfileArn string
		InstanceProfileId  string
	}

	res, err := http.Get("http://" + ec2MetadataAddr + "/latest/meta-data/iam/info")
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&profile); err != nil {
		return "", fmt.Errorf("failed to parse response fetching role: %s", err)
	}

	roleArn := profile.InstanceProfileArn
	arnParts := strings.Split(roleArn, "/")

	if len(arnParts) != 2 {
		return "", fmt.Errorf("bad instance profile arn: %s", roleArn)
	}

	role := arnParts[1]
	return role, nil
}

func fetchSecret(token string, role string, name string) (string, error) {
	events.Log("fetching secret value of %{variable}s from %{address}s using %{role}s role", name, vaultAddr, role)

	var vault struct {
		Data struct {
			Value string `json:"value"`
		} `json:"data"`
	}

	req, _ := http.NewRequest("GET", "http://"+vaultAddr+"/v1/secret/"+role+"/"+name, nil)
	req.Header.Add("X-Vault-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&vault); err != nil {
		return "", fmt.Errorf("failed to parse response fetching %s: %s", name, err)
	}

	if len(vault.Data.Value) == 0 {
		return "", fmt.Errorf("no secret found in vault response for %s", name)
	}

	return vault.Data.Value, nil
}

func getenv(name string, defval string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		value = defval
	}
	return value
}

func usage(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Usage:\n\tsecret-bootstrap [options...] role [vars...] -- command...\nError:\n\t"+format+"\n", args...)
	os.Exit(1)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error:\n\t"+format+"\n", args...)
	os.Exit(1)
}
