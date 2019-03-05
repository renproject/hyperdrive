# Hyperdrive

An experimental consensus algorithm for secure multiparty computations. Inspired by Tendermint.

Built with ❤ by Ren.

## Build Issues

`dep ensure` will fail to pull down the c header files for
ethereum-go. These are the instructions I followed to fix it locally:

```
go get github.com/kardianos/govendor
govendor init
govendor add +e
# Remove the directory that is missing the c dependencies
rm -rf ./vendor/github.com/ethereum/go-ethereum/crypto/secp256k1/
# Add the file and include all files
# https://github.com/kardianos/govendor/issues/247
govendor add github.com/ethereum/go-ethereum/crypto/secp256k1/^
```

### Message signatures with ECDSA

I picked ethereum-go's implementation of ECDSA instead of the built in
go libraries since it provides both build in marshaling/unmarshaling
for the signature and a way to recover the Public Key from the
signature. The reality is, we don't want to check if a given signature
matches a given Public Key. Instead we want to know who signed this
message, which is the Public Key. Then its easy to write logic to
ensure nobody votes twice given a list of Public Keys or a list of
hashed Public Keys, etc...
