package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/segmentio/events"
	_ "github.com/segmentio/events/ecslogs"
	_ "github.com/segmentio/events/text"
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

	wg := sync.WaitGroup{}

	for _, name := range vars {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			value, err := fetchSecret(role, name)
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

func fetchSecret(role string, name string) (string, error) {
	secret := fmt.Sprintf("%s.%s", role, name)
	sess := session.Must(session.NewSession())
	svc := ssm.New(sess, &aws.Config{Region: aws.String("us-west-2")})
	i := &ssm.GetParametersInput{
		Names:          []*string{&secret},
		WithDecryption: aws.Bool(true),
	}
	o, err := svc.GetParameters(i)
	if err != nil {
		return "", fmt.Errorf("Could not get parameters: %s", err)
	}
	if len(o.InvalidParameters) != 0 {
		return "", errors.New("Invalid paramter found")
	}
	if len(o.Parameters) != 1 {
		return "", errors.New("Invalid number of returned paramters")
	}
	return *o.Parameters[0].Value, nil
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
