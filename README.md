# secret-bootstrap
secret-bootstrap is a tool that is used to securely bootstrap an environment
that contains secrets.

# Usage
Running the program is very easy once the source code is built.

The entire secret-bootstrap process when successful is a single command
```
  $ ./secret-bootstrap iam-role.YOUR_FIRST_VARIABLE iam-role.YOUR_SECOND_VARIABLE
 YOUR_FIRST_VARIABLE=heyyyyy YOUR_SECOND_VARIABLE=heyyysecond

```

## Building secret-bootstrap
Fetching and building the application is done easily with `go get`

```
  $ go get github.com/segmentio/secret-bootstrap
```

## Explaining secrets-bootstrap
secret-bootstrap is a small enough program to explain the entire secret
bootstrapping process in a short paragraph!

Each command line arg given to secret-bootstrap is sent to the secure parameter
store provided by AWS.

The result from the secure parameter store is then printed to the command line
with the role information stripped from it.

If secret-bootstrap has troubles loading any of the secrets from the parameter
store, the program will quit with an exit status of 1.

## Registering a secret with vault.
todo in the morning. Robo stage. Robo prod.

