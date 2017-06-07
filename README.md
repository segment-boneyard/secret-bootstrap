# secret-bootstrap
`secret-bootstrap` is a tool that is used to securely bootstrap an environment
that contains secrets.  
It works in a similar fashion to [aws-vault](https://github.com/99designs/aws-vault),
fetching secrets from Vault, setting environment variables, and replacing itself
with the process that needs the secrets.

## Installation
Fetching and building the application is done easily with `go get`:
```
$ go get github.com/segmentio/secret-bootstrap
```

## Usage
Running the program is very easy once the source code is built:
```
$ secret-bootstrap <iam-role> <vars...> -- <command>
```

The arguments are:

- The IAM role is supposed to be a role assumed by the ECS tasks from which
`secret-bootstrap` is running.

- The list of variables are supposed to be stored in Vault, which will be set
as environment variables.

- An arbitrary command to execute and which inherits the environment that
`secret-bootsrap` has set.

## Registering a secret with vault.

Secret-bootstrap can't help you there. Instead read this paperdoc:
https://paper.dropbox.com/doc/Registering-a-secret-with-vault-vUzW7KNMO5RTpZiQfhqxY

