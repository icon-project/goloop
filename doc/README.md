---
home: true
heroText: Goloop Documentation
tagline: Goloop Documentation
actionText: Get Started →
actionLink: /build
---

## Introduction

Support Goloop document site based on [Vuepress](https://v1.vuepress.vuejs.org).

## Preparation

* node install

    **Mac OSX**
    ```
    brew install node
    ```

* yarn install

    **Mac OSX**
    ```
    brew install yarn
    ```

* package install

    ```
    yarn install
    ```

## Scripts

* `doc:dev` : start a development server

    ```
    yarn doc:dev
    ```

* `doc:build` : build dir as a static site
    
    ```
    yarn doc:build
    ```
    - build output is located in `.vuepress/dist`

* `doc:serve` : serve a static site
    
    ```
    yarn doc:serve
    ```
    - serve `.vuepress/dist` dir

* `api:gen` : generate api documents
    
    ```
    yarn api:gen
    ```
    - api documents : `goloop_admin_api.md`, `goloop_cli.md`
    - to use `api:gen`, you need to build `goloop` binary
    
