package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

const (
	Usage   = `./secret-bootstrap secretFile`
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

func FetchSecrets(s SecretFile) (Outfile, error) {
	sess := session.Must(session.NewSession())
	svc := ssm.New(sess, &aws.Config{Region: aws.String("us-west-2")})
	i := &ssm.GetParametersInput{
		Names:          s,
		WithDecryption: aws.Bool(true),
	}
	o, err := svc.GetParameters(i)
	fmt.Println(err)
	fmt.Println(o)

	return nil, nil
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
