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

Note: you get covermerge from here: `go get github.com/loongy/covermerge`
