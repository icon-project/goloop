# Genesis Storage

Genesis storage is zip archive file containing raw genesis transaction and
other referenced data.
This is used for delivery genesis data.

## Entries in the archive

### Genesis transaction

File name is fixed, `genesis.json`.
For detailed information, please refer [Genesis Transaction](genesis_tx.md).

### Genesis data

Data referenced by the genesis transaction should be stored as genesis data.
File name is hex-decimal representation of sha3-256 hash value of the data.
So, it should be 64 lower case hex decimal characters.

## Genesis template

### Introduction

It's hard to make genesis transaction with hash values of other genesis
data, and also include them for genesis data. Genesis transaction including
followed directives is called as **genesis template**.

Some utilities accept genesis template for genesis storage.

### Directives

You may use following directives in values of genesis transaction.

:::  v-pre
* `{{hash:<file>}}` <br>
  It will be replaced with hex decimals of hash value of
  specified file. And the file will be included into the storage automatically.
  
  Example
  ```json
  "hash:{{hash:governance.zip}}"
  ```
  ```json
  "hash:0x810b7af78caf4bc70a660f0df51e42baf91d4de5b2328de0e83dfc56fd70a6cb"
  ```
  
* `{{read:<file>}}` <br>
  It will be replaced with hex decimals of file content itself.
  It's designated to use in `content` tag.
  
  Example
  ```json
  "{{read:governance.zip}}"
  ```
  ```json
  "0xaabbccdd...."
  ```
  
* `{{ziphash:<dir>}}` <br>
  It's very similar to `{{hash:<file>}}` except that it accepts a directory
  for input and makes a zip archive before storing and hashing.
  
* `{{zip:<dir>}}` <br>
  It's very similar to `{{read:<file>}}` except that it accepts a directory
  for input and makes a zip archive for reading.
:::

You may use template in values of genesis transaction. You may use this
scheme for making genesis storage from genesis transaction while it includes
other genesis data.

**Note:**
If you refer same directory in different positions, it may returns different
hash value or bytes.

