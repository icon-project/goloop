# Goloop
[![codecov](https://codecov.io/gh/icon-project/goloop/branch/master/graph/badge.svg?token=5DUTFEGMFY)](https://codecov.io/gh/icon-project/goloop)

Official Blockchain Architecture Implementation of the ICON 2.0 Protocol.

Welcome to the Goloop github repository. Goloop is a smart contract enabled enterprise-grade blockchain software written in Go with many unique features providing a secure, immutable and scalable environment to develop decentralized applications.

This github repository contains the source code of the Goloop software, which will serve as base modules for enterprise blockchains as well as the implementation base for ICON 2.0 protocol.

## Introduction

* General
  - [Build Guide](doc/build.md)
  - [Tutorial](doc/tutorial.md)
* API Documents
  - [JSON RPC v3](doc/jsonrpc_v3.md)
  - [JSON RPC IISS Extension](doc/iiss_extension.md)
  - [JSON RPC BTP Extension](doc/btp_extension.md)
  - [JSON RPC BTP2 Extension](doc/btp2_extension.md)
  - [ChainSCORE APIs](doc/icon_chainscore_api.md)
* Others
  - [`goloop` command line reference](doc/goloop_cli.md)
  - [Genesis Transaction](doc/genesis_tx.md)
  - [Genesis Storage](doc/genesis_storage.md)
  - [ICON Config](doc/icon_config.md)

## Contribution Guidelines

There are two branches in the repository. "master" is the development branch for ICON 2.0 and the "base" is the branch shared with enterprises and public blockchains. Therefore, the updates only focused on the base modules will be committed to the "base" branch, while the economics and governance logics focused updates are committed to the "master" branch.

We will update more detailed guidelines to collaborate with community developers.

## License

This project is available under the [Apache License, Version 2.0](LICENSE).

## Community

The following links are the ICON community developer chat channel, developer website, and community forum. More detailed information regarding Goloop will be updated on the links below.

- [ICON Community Developer Chat Channel](https://t.me/icondevs)
- [ICON Developer Portal](https://www.icondev.io/)
- [ICON Community Forum](https://forum.icon.community/)
