#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <projectDir> <mainClassFullName>"
    exit 1
fi

die () {
    echo "Error: $*"
    exit 1
}

PROJECT=$1
MAIN_CLASS=$2

TOPDIR=$(dirname $(realpath $0))
PARENTDIR=$TOPDIR/..

VERSION=$(grep "^VERSION=" $PARENTDIR/gradle.properties | cut -d= -f2)
CLASSPATH=$PARENTDIR/api/build/libs/api-$VERSION.jar
if [ ! -f $CLASSPATH ]; then
    CLASSPATH=$PARENTDIR/api/build/libs/api-$VERSION-SNAPSHOT.jar
    if [ ! -f $CLASSPATH ]; then
        die "Cannot find $CLASSPATH"
    fi
fi

# Determine the Java command to use to start the JVM
if [[ -n "$JAVA_HOME" ]] && [[ -x "$JAVA_HOME/bin/javac" ]]; then
    JAVAC="$JAVA_HOME/bin/javac"
    JAR="$JAVA_HOME/bin/jar"
else
    JAVAC="javac"
    JAR="jar"
fi
JAVAC_OPTIONS='-parameters --release 11'

# cd to project and clean the previous build
cd $TOPDIR/$PROJECT || die "Could not change directory to $PROJECT"
echo "Cleaning the build folder..."
rm -fr "./build"

# compile
echo "Compiling the source code..."
SOURCE_FILES=$(find ./src -name *.java)
$JAVAC $JAVAC_OPTIONS -cp $CLASSPATH -d "./build" $SOURCE_FILES || exit 3

# assemble the jar
echo "Assembling the final jar..."
cd "./build"
mkdir ./META-INF/
echo "Main-Class: $MAIN_CLASS" > "./META-INF/MANIFEST.MF"
$JAR -cfm "dapp.jar" "./META-INF/MANIFEST.MF" .
cd ..

# done!
echo "Success!"
echo "The jar has been generated at $(realpath ./build/dapp.jar)"
