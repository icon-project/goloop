GRADLE=./gradlew

javaexec:
	@ \
	echo "[#] Building Java executor" ; \
	export GRADLE_USER_HOME=.gradle ; \
	rm -rf exec/build/native ; \
	$(GRADLE) -Prelease app:execman:distZip

test:
	@ \
	echo "[#] Running javaee test" ; \
	export GRADLE_USER_HOME=.gradle ; \
	$(GRADLE) test

clean:
	@ \
	echo "[#] Cleaning javaee" ; \
	$(GRADLE) clean

.PHONY: test clean
