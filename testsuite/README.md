# TEST Suite

A collection of test suite for goloop.
It includes test cases for continuous integration system.

## Requirements

You need to install OpenJDK 11 version. Visit [OpenJDK.net](http://openjdk.java.net/) for prebuilt binaries.
Or you can install a proper OpenJDK package from your OS vendors.

In macOS:
```
$ brew tap AdoptOpenJDK/openjdk
$ brew cask install adoptopenjdk11
```

In Linux (Ubuntu 18.04):
```
$ sudo apt install openjdk-11-jdk
```

## Run

```
$ ./gradlew <target>
```

For specific test cases.
```
$ ./gradlew <target> --tests <test pattern>
```

### Available targets

| Target         | Description                                          |
|:---------------|:-----------------------------------------------------|
| testPyScore    | Test features for Python SCORE.                      |
| testPyGov      | Test governance SCORE written in Python.             |
| testJavaScore  | Test features for Java SCORE.                        |
| testJavaGov    | Test governance SCORE written in Java.               |


### Options

Using options.
```
$ ./gradlew [-D<variable>=<value>...] <target>
```

Available options.

| Option      | Targets   | Description                                    |
|:------------|:----------|:-----------------------------------------------|
| `AUDIT`     | testPyGov | `true` for testing SCOREs with AUDIT feature   |
| `NO_SERVER` | all       | `true` for disabling auto start of `gochain`.  |
| `USE_DOCKER`| all       | `true` for enabling docker container for test. |

To use other nodes than `gochain`, start the servers first, then define
`NO_SERVER` as `true`.

To run docker container for the node, set `USE_DOCKER` as `true`.
 
Set `AUDIT` to `true` to run audit specific feature tests in governance.

## Structure

### Directory structure

| Directory           | Description                                    |
|:--------------------|:-----------------------------------------------|
| data/genesisStorage | Genesis files & governance SCOREs              |
| data/scores         | SCOREs related with test cases                 |
| data/config         | gochain configurations for the target          |
| data/chainenv       | Chain environment property files for the target|
| data/dockerenv      | Docker environment files for the target        |
| java                | Java sources related with test cases           |
| gradle              | Gradle wrapper directory                       |
| build               | Build output directory                         |
| out                 | Test output directory                          |

### Packages

| Package                     | Description                 |
|:----------------------------|:----------------------------|
| foundation.icon.test.cases  | Test case classes           |
| foundation.icon.test.common | Common classes              |
| foundation.icon.test.scores | Wrapping classes for SCOREs |

### Test cases

All test cases are written in JUnit 5.
> Refer [https://junit.org/junit5/] for JUnit.

#### Environment files

Before it executes test cases, it loads environment properties from
the file specified by environment variable `CHAIN_ENV`
(default value is `"./data/env.properties"`).
It's accessible through `foundation.icon.test.common.Env`.

#### Tags

Test cases are categorized into specific targets.
To identify test cases, following tags are used for each target.

| Target         | Tags            |
|:---------------|:----------------|
| testPyScore    | TAG_PY_SCORE    |
| testPyGov      | TAG_PY_GOV      |
| testJavaScore  | TAG_JAVA_SCORE  |
| testJavaGov    | TAG_JAVA_GOV    |

> Example of `TAG_PY_SCORE`
```java
@Tag(Constants.TAG_PY_SCORE)
@Test
void testPythonToPython() throws Exception {
    // test codes.
}
```
