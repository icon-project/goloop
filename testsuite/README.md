# TEST Suite

## About

This is test suite for goloop.
It includes test cases for continuous integration system.

## Requirements

### JDK 11

It's tested with OpenJDK 11.0.4.
> Refer [https://openjdk.java.net/] for OpenJDK

#### Installation in macOS

```
brew tap AdoptOpenJDK/openjdk
brew cask install adoptopenjdk11
```
> Refer [https://brew.sh/] for `brew`

## Run

```
# ./gradlew build
# ./gradlew <target>
```

For specific test case
```
# ./gradlew <target> --tests <test pattern>
```

### Available targets

| Target         | Description                                          |
|:---------------|:-----------------------------------------------------|
| testPyScore    | Test features for python score.                      |
| testPyGov      | Test chain score with python score.                  |
| testJavaScore  | Test features for java score.                        |
| testInterScore | Test inter operation between java and python scores. |


### Options

Using options.
```
# ./gradlew [-D<variable>=<value>...] <target>
```

Available options

| Option      | Targets   | Description                                    |
|:------------|:----------|:-----------------------------------------------|
| AUDIT       | testPyGov | `true` for testing scores with AUDIT feature   |
| NO_SERVER   | all       | `true` for disabling auto start `gochain`.     |
| USE_DOCKER  | all       | `true` for enabling docker container for test. |

To use other nodes than `gochain`, start the servers first, then define
`NO_SERVER` as `true`.

To run docker container for the node, set both `NO_SERVER` and `USE_DOCKER`
 as `true`
 
Set `AUDIT` to `true` to run audit specific feature tests in governance.

## Structure

### Directory structure

| Directory           | Description                                    |
|:--------------------|:-----------------------------------------------|
| data/genesisStorage | Genesis file & governance score                |
| data/scores         | SCOREs related with test cases                 |
| data/config         | gochain configurations for the target          |
| data/chainenv       | Chain environment property file for the target |
| data/dockerenv      | Docker environment file for the target         |
| java                | Java sources related with test cases           |
| gradle              | Gradle class directory                         |
| build               | Build output directory                         |
| out                 | Test output directory                          |

### Packages

| Package                     | Description                 |
|:----------------------------|:----------------------------|
| foundation.icon.test.cases  | Test case classes           |
| foundation.icon.test.common | Common classes              |
| foundation.icon.test.scores | Wrapping classes for SCOREs |

### Test case

All test cases are written in JUnit 5.
> Refer [https://junit.org/junit5/] for JUnit.

#### Environment file

Before it executes test cases, it loads environment properties from
the file specified by environment variable `CHAIN_ENV`
 ( default value is `"./data/env.properties"`).
It's accessible through `foundation.icon.test.common.Env`.

#### Tags
They are categorized into specific targets.
To identify test cases, following tags are used for each target.

| Target         | Tags            |
|:---------------|:----------------|
| testPyScore    | TAG_PY_SCORE    |
| testPyGov      | TAG_PY_SCORE    |
| testJavaScore  | TAG_JAVA_SCORE  |
| testInterScore | TAG_INTER_SCORE |

> Example of `TAG_PY_SCORE`
```java
@Tag(Constants.TAG_PY_SCORE)
@Test
void testPythonToPython() throws Exception {
    // test codes.
}
```
