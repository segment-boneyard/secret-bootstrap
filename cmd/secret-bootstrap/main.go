package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

const (
	Usage   = `./secret-bootstrap <iam_role> <SECRET_NAMEs...>`
	OutName = "secrets.sources"
)

var (
	EnvLine = "%s=\"%s\""
)

type SecretFile []*string

type Outfile []string

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

func FetchSecrets(s SecretFile) (string, error) {
	var (
		secret string
	)
	sess := session.Must(session.NewSession())
	svc := ssm.New(sess, &aws.Config{Region: aws.String("us-west-2")})
	i := &ssm.GetParametersInput{
		Names:          s,
		WithDecryption: aws.Bool(true),
	}
	o, err := svc.GetParameters(i)
	if err != nil {
		return "", err
	}
	if len(o.InvalidParameters) != 0 {
		return "", errors.New("parameter_store_missing_parameter")
	}
	for _, param := range o.Parameters {
		sp := strings.Split(*param.Name, ".")
		if len(sp) != 2 {
			return "", errors.New("parameter_store_invalid_parameter")
		}

		secret = fmt.Sprintf("%s %s=%s", secret, sp[1], *param.Value)
	}
	return secret, nil
}

func main() {
	var (
		s SecretFile
	)
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		printUsageAndExit()
	}

	role := ""
	for i, _ := range args {
		if i == 0 {
			role = args[i]
			continue
		}
		fullArg := fmt.Sprintf("%s.%s", role, args[i])
		s = append(s, &fullArg)
	}

	envs, err := FetchSecrets(s)
	if err != nil {
		log.Fatalf("Could not fetch secrets: %s", err)
	}
	fmt.Println(envs)
}
