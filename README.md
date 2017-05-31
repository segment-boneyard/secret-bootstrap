# secret-bootstrap
secret-bootstrap is a tool that is used to securely bootstrap an environment
that contains secrets.

It's important to be aware that secret-bootstrap, when successfully run, will
write secrets to a file called `secrets.sources`.

# Usage
Running the program is very easy once the source code is built.

The entire secret-bootstrap process when successful should look something like
this:
```
  $ ./secret-bootstrap secretfile.json
  $ source secrets.source
  $ ./my-perfect-app
```

## Building secret-bootstrap
Fetching and building the application is done easily with `go get`

```
  $ go get github.com/segmentio/secret-bootstrap
```

## Making a secretfile.json
The secretfile format is not meant to be complex. It is a map of names of
environment variables you wish to be set, that each map to a secret that
is stored in vault.

Below is an example secretfile
```
  {
    "MY_ENV_VARIABLE":"bastion/foo",
    "MY_OTHER_VARIABLE":"bastion/hello",
  }
```

## Registering a secret with vault.

